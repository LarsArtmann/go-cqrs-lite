package saga

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// ExecuteStep runs the current step of a saga instance.
func (r *Runner) ExecuteStep(ctx context.Context, instanceID id.AggregateID) error {
	state, err := r.store.Load(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("load saga %s: %w", instanceID, err)
	}

	instance, err := r.hydrate(state)
	if err != nil {
		return err
	}

	if instance.Status != StatusRunning && instance.Status != StatusPending {
		return fmt.Errorf("saga %s is not running (status=%s)", instanceID, instance.Status)
	}

	if instance.CurrentStep >= len(instance.Steps) {
		instance.Status = StatusCompleted
		instance.UpdatedAt = time.Now()
		return r.store.Save(ctx, &instance.State)
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
		instance.ErrMsg = err.Error()
		instance.UpdatedAt = time.Now()

		if instance.CurrentStep > 0 {
			r.logError(
				"step failed, compensating",
				"id",
				instanceID,
				"step",
				step.Name,
				"error",
				err,
			)
			instance.Status = StatusCompensating
			if saveErr := r.store.Save(ctx, &instance.State); saveErr != nil {
				return fmt.Errorf("save compensating status: %w", saveErr)
			}
			return r.compensate(ctx, instance)
		}

		instance.Status = StatusFailed
		if saveErr := r.store.Save(ctx, &instance.State); saveErr != nil {
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

	if err := r.store.Save(ctx, &instance.State); err != nil {
		return fmt.Errorf("save step completion: %w", err)
	}

	r.logInfo(
		"step completed",
		"id",
		instance.ID,
		"step",
		step.Name,
		"current",
		instance.CurrentStep,
	)
	return nil
}

// hydrate assembles a runtime Instance from a persisted State.
func (r *Runner) hydrate(state *State) (*Instance, error) {
	r.mu.RLock()
	def, ok := r.registry[state.SagaType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("saga %s: %w", state.SagaType, ErrSagaNotRegistered)
	}

	instance := &Instance{
		State: *state,
		Steps: def.Steps(),
	}

	if state.ErrMsg != "" {
		instance.Err = errors.New(state.ErrMsg)
	}

	return instance, nil
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
