package command

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound error = errorfamily.NewRejection(
	"command.handler_not_found",
	"handler not found for command",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed error = errorfamily.NewInfrastructure(
	"command.dispatcher_closed",
	"command dispatcher is closed",
)

// ErrEmptyCommandType is returned by New when the command type is empty.
var ErrEmptyCommandType error = errorfamily.NewRejection(
	"command.empty_command_type",
	"command type is required",
)

// ErrNilStreamID is returned by New when the stream ID is zero.
var ErrNilStreamID error = errorfamily.NewRejection(
	"command.nil_stream_id",
	"stream ID is required",
)

// Deprecated: use ErrNilStreamID.
var ErrNilAggregateID error = ErrNilStreamID

// ErrTypeAssertion is returned when a command cannot be type-asserted to the expected type.
var ErrTypeAssertion error = errorfamily.NewRejection(
	"command.type_assertion",
	"command type assertion failed",
)

// ErrEmptyStreamType is returned when a stream type is empty.
var ErrEmptyStreamType error = errorfamily.NewRejection(
	"command.empty_stream_type",
	"stream type is required",
)

// Deprecated: use ErrEmptyStreamType.
var ErrEmptyAggregateType error = ErrEmptyStreamType

// ErrDuplicateCommand is returned when a command with the same ID already exists.
var ErrDuplicateCommand error = errorfamily.NewConflict(
	"command.duplicate",
	"command with this ID already exists",
)

// ErrCommandNotFound is returned when a command is not found.
var ErrCommandNotFound error = errorfamily.NewRejection(
	"command.not_found",
	"command not found",
)

// ErrStoreClosed is returned when the command store is closed.
var ErrStoreClosed error = errorfamily.NewInfrastructure(
	"command.store_closed",
	"command store is closed",
)
