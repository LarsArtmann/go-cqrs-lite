package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Runner manages the lifecycle of saga instances.
type Runner struct {
	store      Store
	dispatcher CommandDispatcher
	registry   map[string]Definition
	mu         sync.RWMutex
	config     runnerConfig
}

// NewRunner creates a new saga runner.
func NewRunner(store Store, dispatcher CommandDispatcher, opts ...RunnerOption) *Runner {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Runner{
		store:      store,
		dispatcher: dispatcher,
		registry:   make(map[string]Definition),
		config:     cfg,
	}
}

// Register adds a saga definition to the runner.
func (r *Runner) Register(def Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def == nil {
		return fmt.Errorf("definition is nil: %w", ErrSagaNotFound)
	}

	sagaType := def.SagaType()
	if sagaType == "" {
		return fmt.Errorf("saga type is empty: %w", ErrSagaNotFound)
	}

	if _, exists := r.registry[sagaType]; exists {
		return fmt.Errorf("saga %s: %w", sagaType, ErrSagaAlreadyExists)
	}

	r.registry[sagaType] = def
	return nil
}

// Start begins a new saga instance with the given initial command.
func (r *Runner) Start(ctx context.Context, sagaType string, initialCommand command.Command) (*Instance, error) {
	r.mu.RLock()
	def, ok := r.registry[sagaType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("saga %s: %w", sagaType, ErrSagaNotRegistered)
	}

	instance := &Instance{
		ID:          id.NewAggregateID(),
		SagaType:    sagaType,
		Status:      StatusPending,
		CurrentStep: 0,
		Steps:       def.Steps(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.store.Save(ctx, instance); err != nil {
		return nil, fmt.Errorf("save saga instance: %w", err)
	}

	instance.Status = StatusRunning
	if err := r.store.Save(ctx, instance); err != nil {
		return nil, fmt.Errorf("update saga status: %w", err)
	}

	r.logInfo("saga started", "type", sagaType, "id", instance.ID)

	if initialCommand != nil {
		if err := r.dispatcher.Dispatch(ctx, initialCommand); err != nil {
			instance.Status = StatusFailed
			instance.Err = err
			_ = r.store.Save(ctx, instance)
			r.logError("initial command failed", "type", sagaType, "id", instance.ID, "error", err)
			return instance, fmt.Errorf("dispatch initial command: %w", err)
		}
	}

	return instance, nil
}

// ExecuteStep runs the current step of a saga instance.
func (r *Runner) ExecuteStep(ctx context.Context, instanceID id.AggregateID) error {
	instance, err := r.store.Load(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("load saga %s: %w", instanceID, err)
	}

	if instance.Status != StatusRunning && instance.Status != StatusPending {
		return fmt.Errorf("saga %s is not running (status=%s)", instanceID, instance.Status)
	}

	if instance.CurrentStep >= len(instance.Steps) {
		instance.Status = StatusCompleted
		instance.UpdatedAt = time.Now()
		return r.store.Save(ctx, instance)
	}

	step := instance.Steps[instance.CurrentStep]

	var stepCtx context.Context
	var cancel context.CancelFunc
	if step.Timeout > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
	} else {
		stepCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := step.Action(stepCtx, instance.ID)
	if cmd == nil {
		return fmt.Errorf("step %s returned nil command", step.Name)
	}

	err = r.dispatchWithRetry(stepCtx, cmd)
	if err != nil {
		instance.Err = err
		instance.UpdatedAt = time.Now()

		if instance.CurrentStep > 0 {
			r.logError("step failed, compensating", "id", instanceID, "step", step.Name, "error", err)
			instance.Status = StatusCompensating
			if saveErr := r.store.Save(ctx, instance); saveErr != nil {
				return fmt.Errorf("save compensating status: %w", saveErr)
			}
			return r.compensate(ctx, instance)
		}

		instance.Status = StatusFailed
		if saveErr := r.store.Save(ctx, instance); saveErr != nil {
			return fmt.Errorf("save failed status: %w", saveErr)
		}
		r.logError("step failed", "id", instanceID, "step", step.Name, "error", err)
		return fmt.Errorf("step %s: %w", step.Name, err)
	}

	instance.CurrentStep++
	instance.UpdatedAt = time.Now()

	if instance.CurrentStep >= len(instance.Steps) {
		instance.Status = StatusCompleted
		r.logInfo("saga completed", "id", instance.ID, "type", instance.SagaType)
	}

	if err := r.store.Save(ctx, instance); err != nil {
		return fmt.Errorf("save step completion: %w", err)
	}

	r.logInfo("step completed", "id", instance.ID, "step", step.Name, "current", instance.CurrentStep)
	return nil
}

// dispatchWithRetry attempts to dispatch a command with exponential backoff.
func (r *Runner) dispatchWithRetry(ctx context.Context, cmd command.Command) error {
	var err error
	delay := r.config.retryDelay

	for attempt := 0; attempt <= r.config.maxRetries; attempt++ {
		err = r.dispatcher.Dispatch(ctx, cmd)
		if err == nil {
			return nil
		}

		if attempt < r.config.maxRetries {
			if !event.IsRetryable(err) {
				return err
			}

			select {
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * r.config.retryMultiplier)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return err
}

// compensate runs compensation for all completed steps in reverse order.
func (r *Runner) compensate(ctx context.Context, instance *Instance) error {
	for i := instance.CurrentStep - 1; i >= 0; i-- {
		step := instance.Steps[i]
		if step.Compensate == nil {
			continue
		}

		cmd := step.Compensate(ctx, instance.ID)
		if cmd == nil {
			continue
		}

		if err := r.dispatcher.Dispatch(ctx, cmd); err != nil {
			r.logError("compensate step failed", "id", instance.ID, "step", step.Name, "error", err)
			instance.Status = StatusFailed
			instance.Err = fmt.Errorf("compensate step %s: %w", step.Name, err)
			instance.UpdatedAt = time.Now()
			_ = r.store.Save(ctx, instance)
			return instance.Err
		}

		r.logInfo("compensated step", "id", instance.ID, "step", step.Name)
	}

	instance.Status = StatusFailed
	instance.Err = ErrStepFailed
	instance.UpdatedAt = time.Now()

	r.logInfo("compensation completed", "id", instance.ID, "type", instance.SagaType)

	if err := r.store.Save(ctx, instance); err != nil {
		return fmt.Errorf("save compensated status: %w", err)
	}

	return ErrStepFailed
}

func (r *Runner) logInfo(msg string, attrs ...any) {
	if r.config.logger != nil {
		r.config.logger.Info(msg, attrs...)
	}
}

func (r *Runner) logError(msg string, attrs ...any) {
	if r.config.logger != nil {
		r.config.logger.Error(msg, attrs...)
	}
}
