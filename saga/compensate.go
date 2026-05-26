package saga

import (
	"context"
	"fmt"
	"time"
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
