package saga

import "github.com/larsartmann/go-cqrs-lite/core/event"

var (
	ErrSagaNotFound       = event.NewRejection("saga.not_found", "saga not found")
	ErrSagaAlreadyExists  = event.NewConflict("saga.already_exists", "saga already exists")
	ErrStepFailed         = event.NewInfrastructure("saga.step_failed", "saga step failed")
	ErrCompensationFailed = event.NewInfrastructure(
		"saga.compensation_failed",
		"saga compensation failed",
	)
	ErrSagaTimeout       = event.NewTransient("saga.timeout", "saga step timed out")
	ErrSagaNotRegistered = event.NewRejection("saga.not_registered", "saga type not registered")
)
