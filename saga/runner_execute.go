package saga

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// ExecuteStep runs the current step of a saga instance.
func (r *Runner) ExecuteStep(ctx context.Context, instanceID id.AggregateID) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "saga.step.execute",
		trace.SpanKindInternal,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrAggregateID, instanceID.String()),
		),
	)
	defer span.End()

	state, err := r.store.Load(ctx, instanceID)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "saga.load_failed", "load saga "+instanceID.String())
	}

	instance, err := r.hydrate(state)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	if instance.Status != StatusRunning && instance.Status != StatusPending {
		err := event.NewRejection(
			"saga.not_running",
			"saga "+instanceID.String()+" is not running (status="+string(instance.Status)+")",
		)
		cqrsotel.RecordError(span, err)

		return err
	}

	if instance.CurrentStep >= len(instance.Steps) {
		instance.Status = StatusCompleted
		instance.UpdatedAt = time.Now()

		err := r.store.Save(ctx, &instance.State)
		if err != nil {
			cqrsotel.RecordError(span, err)
		}

		return err
	}

	step := instance.Steps[instance.CurrentStep]

	span.SetAttributes(
		attribute.String(cqrsotel.AttrSagaStepName, step.Name),
		attribute.Int(cqrsotel.AttrSagaStep, instance.CurrentStep),
	)

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
		err := event.NewRejection("saga.nil_command", "step "+step.Name+" returned nil command")
		cqrsotel.RecordError(span, err)

		return err
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
				return event.WrapInfrastructure(saveErr, "saga.save_compensating_failed", "save compensating status")
			}
			return r.compensate(ctx, instance)
		}

		instance.Status = StatusFailed
		if saveErr := r.store.Save(ctx, &instance.State); saveErr != nil {
			return event.WrapInfrastructure(saveErr, "saga.save_failed_status_failed", "save failed status")
		}
		r.logError("step failed", "id", instanceID, "step", step.Name, "error", err)
		return event.WrapInfrastructure(err, "saga.step_failed", "step "+step.Name+" failed")
	}

	instance.CurrentStep++
	instance.UpdatedAt = time.Now()

	if instance.CurrentStep >= len(instance.Steps) {
		instance.Status = StatusCompleted
		r.logInfo("saga completed", "id", instance.ID, "type", instance.SagaType)
	}

	if err := r.store.Save(ctx, &instance.State); err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "saga.save_completion_failed", "save step completion")
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
		return nil, event.WrapRejection(
			ErrSagaNotRegistered,
			"saga.not_registered",
			"saga "+state.SagaType+" not registered",
		)
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
