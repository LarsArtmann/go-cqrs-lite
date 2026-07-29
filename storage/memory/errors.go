package memory

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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

// withReadLock centralises the wrapClosed + RLock + defer RUnlock preamble for
// read-side MemoryStore methods. It is a top-level generic function because Go
// does not permit generic methods; the store is passed explicitly. The message
// is pre-formatted by the caller so both plain and formatted close errors share
// one path.
func withReadLock[T any](
	s *MemoryStore,
	code, msg string,
	fn func() (T, error),
) (T, error) {
	var zero T

	if err := wrapClosed(s.CheckClosed(event.ErrStoreClosed), code, msg); err != nil {
		return zero, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn()
}
