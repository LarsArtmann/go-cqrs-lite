package saga

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

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

	if err := r.store.Save(ctx, &instance.State); err != nil {
		return event.WrapInfrastructure(err, "saga.save_compensated_failed", "save compensated status")
	}

	return ErrStepFailed
}
