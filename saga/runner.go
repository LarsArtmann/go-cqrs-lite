package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
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
func (r *Runner) Start(
	ctx context.Context,
	sagaType string,
	initialCommand command.Command,
) (*Instance, error) {
	r.mu.RLock()
	def, ok := r.registry[sagaType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("saga %s: %w", sagaType, ErrSagaNotRegistered)
	}

	state := State{
		ID:          id.NewAggregateID(),
		SagaType:    sagaType,
		Status:      StatusPending,
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.store.Save(ctx, &state); err != nil {
		return nil, fmt.Errorf("save saga state: %w", err)
	}

	state.Status = StatusRunning
	if err := r.store.Save(ctx, &state); err != nil {
		return nil, fmt.Errorf("update saga status: %w", err)
	}

	instance := &Instance{
		State: state,
		Steps: def.Steps(),
	}

	r.logInfo("saga started", "type", sagaType, "id", instance.ID)

	if initialCommand != nil {
		if err := r.dispatcher.Dispatch(ctx, initialCommand); err != nil {
			instance.Status = StatusFailed
			instance.Err = err
			instance.ErrMsg = err.Error()
			instance.UpdatedAt = time.Now()
			_ = r.store.Save(ctx, &instance.State)
			r.logError("initial command failed", "type", sagaType, "id", instance.ID, "error", err)
			return instance, fmt.Errorf("dispatch initial command: %w", err)
		}
	}

	return instance, nil
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
