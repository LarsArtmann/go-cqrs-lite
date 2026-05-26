package saga

import "errors"

var (
	ErrSagaNotFound       = errors.New("saga not found")
	ErrSagaAlreadyExists  = errors.New("saga already exists")
	ErrStepFailed         = errors.New("saga step failed")
	ErrCompensationFailed = errors.New("saga compensation failed")
	ErrSagaTimeout        = errors.New("saga step timed out")
	ErrSagaNotRegistered  = errors.New("saga type not registered")
)
