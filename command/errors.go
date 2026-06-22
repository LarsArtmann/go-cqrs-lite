package command

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound = event.NewRejection(
	"command.handler_not_found",
	"handler not found for command",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = event.NewInfrastructure(
	"command.dispatcher_closed",
	"command dispatcher is closed",
)

// ErrEmptyCommandType is returned by New when the command type is empty.
var ErrEmptyCommandType = event.NewRejection(
	"command.empty_command_type",
	"command type is required",
)

// ErrNilAggregateID is returned by New when the aggregate ID is zero.
var ErrNilAggregateID = event.NewRejection(
	"command.nil_aggregate_id",
	"aggregate ID is required",
)

// ErrTypeAssertion is returned when a command cannot be type-asserted to the expected type.
var ErrTypeAssertion = event.NewRejection(
	"command.type_assertion",
	"command type assertion failed",
)

// ErrEmptyAggregateType is returned when an aggregate type is empty.
var ErrEmptyAggregateType = event.NewRejection(
	"command.empty_aggregate_type",
	"aggregate type is required",
)

// ErrDuplicateCommand is returned when a command with the same ID already exists.
var ErrDuplicateCommand = event.NewConflict(
	"command.duplicate",
	"command with this ID already exists",
)

// ErrCommandNotFound is returned when a command is not found.
var ErrCommandNotFound = event.NewRejection(
	"command.not_found",
	"command not found",
)

// ErrStoreClosed is returned when the command store is closed.
var ErrStoreClosed = event.NewInfrastructure(
	"command.store_closed",
	"command store is closed",
)
