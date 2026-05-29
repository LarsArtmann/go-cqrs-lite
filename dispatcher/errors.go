package dispatcher

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrHandlerNotFound is returned when no handler is registered for a type.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"dispatcher.handler_not_found",
	"handler not found",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"dispatcher.dispatcher_closed",
	"dispatcher is closed",
)

// ErrHandlerAlreadyRegistered is returned when a handler is already registered for a type.
var ErrHandlerAlreadyRegistered = errorfamily.NewConflict(
	"dispatcher.handler_already_registered",
	"handler already registered for type",
)
