package command

import "github.com/cockroachdb/errors"

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound = errors.New("handler not found for command")

// ErrCommandValidation is returned when command validation fails.
var ErrCommandValidation = errors.New("command validation failed")

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errors.New("command dispatcher is closed")
