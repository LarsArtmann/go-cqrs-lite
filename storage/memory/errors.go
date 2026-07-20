package memory

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrHandlerNil is returned when a nil handler is passed to Subscribe or SubscribeAll.
var ErrHandlerNil = errorfamily.NewRejection(
	"memory.handler_nil",
	"handler must not be nil",
)

// wrapClosed returns nil when err is nil, otherwise wraps it as an
// Infrastructure error with the given code and message. Centralises the
// CheckClosed → errorfamily.WrapInfrastructure boilerplate used by every
// memory store method.
func wrapClosed(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}

// wrapClosedf is the format-args variant of wrapClosed.
func wrapClosedf(err error, code, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return errorfamily.Wrapf(err, errorfamily.Infrastructure, code, format, args...)
}
