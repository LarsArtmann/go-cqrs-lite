package saga

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

func (r *Runner) compensate(ctx context.Context, instance *Instance) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "saga.compensate",
		trace.SpanKindInternal,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrSagaType, instance.SagaType),
			attribute.String(cqrsotel.AttrAggregateID, instance.ID.String()),
		),
	)
	defer span.End()

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
			cqrsotel.RecordError(span, err)
			r.logError("compensate step failed", "id", instance.ID, "step", step.Name, "error", err)
			instance.Status = StatusFailed
			instance.Err = event.WrapInfrastructure(err, "saga.compensate_step_failed", "compensate step "+step.Name)
			instance.ErrMsg = instance.Err.Error()
			instance.UpdatedAt = time.Now()
			_ = r.store.Save(ctx, &instance.State)
			return instance.Err
		}

		r.logInfo("compensated step", "id", instance.ID, "step", step.Name)
	}

	instance.Status = StatusFailed
	instance.Err = ErrStepFailed
	instance.ErrMsg = ErrStepFailed.Error()
	instance.UpdatedAt = time.Now()

	r.logInfo("compensation completed", "id", instance.ID, "type", instance.SagaType)

	if err := r.saveSagaState(
		ctx,
		span,
		&instance.State,
		"saga.save_compensated_failed",
		"save compensated status",
	); err != nil {
		return err
	}

	return ErrStepFailed
}
