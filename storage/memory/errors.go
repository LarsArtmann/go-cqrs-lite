package memory

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrHandlerNil is returned when a nil handler is passed to Subscribe or SubscribeAll.
var ErrHandlerNil = errorfamily.NewRejection(
	"memory.handler_nil",
	"handler must not be nil",
)
