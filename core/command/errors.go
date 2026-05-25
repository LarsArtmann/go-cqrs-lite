package command

import errorfamily "github.com/larsartmann/go-error-family"

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"command.handler_not_found",
	"handler not found for command",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"command.dispatcher_closed",
	"command dispatcher is closed",
)

// ErrEmptyCommandType is returned by New when the command type is empty.
var ErrEmptyCommandType = errorfamily.NewRejection(
	"command.empty_command_type",
	"command type is required",
)

// ErrNilAggregateID is returned by New when the aggregate ID is zero.
var ErrNilAggregateID = errorfamily.NewRejection(
	"command.nil_aggregate_id",
	"aggregate ID is required",
)

// ErrTypeAssertion is returned when a command cannot be type-asserted to the expected type.
var ErrTypeAssertion = errorfamily.NewCorruption(
	"command.type_assertion",
	"command type assertion failed",
)
