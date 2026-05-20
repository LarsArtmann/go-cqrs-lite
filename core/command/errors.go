package command

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound = errors.New("handler not found for command")

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errors.New("command dispatcher is closed")

// ErrEmptyCommandType is returned by New when the command type is empty.
var ErrEmptyCommandType = errors.New("command type is required")

// ErrNilAggregateID is returned by New when the aggregate ID is zero.
var ErrNilAggregateID = errors.New("aggregate ID is required")

// ErrTypeAssertion is returned when a command cannot be type-asserted to the expected type.
var ErrTypeAssertion = errors.New("command type assertion failed")

func init() { //nolint:gochecknoinits // registers error classifications for cross-package Classify()
	event.RegisterClassification(ErrHandlerNotFound, event.Rejection)
	event.RegisterClassification(ErrDispatcherClosed, event.Infrastructure)
	event.RegisterClassification(ErrEmptyCommandType, event.Rejection)
	event.RegisterClassification(ErrNilAggregateID, event.Rejection)
	event.RegisterClassification(ErrTypeAssertion, event.Corruption)
}
