package query

import errorfamily "github.com/larsartmann/go-error-family"

// ErrHandlerNotFound is returned when no handler is registered for a query type.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"query.handler_not_found",
	"no handler registered for query",
)

// ErrQueryNotSupported is an alias for ErrHandlerNotFound.
// Deprecated: Use ErrHandlerNotFound for consistency with command.Dispatcher.
var ErrQueryNotSupported = ErrHandlerNotFound

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"query.dispatcher_closed",
	"query dispatcher is closed",
)

// ErrEmptyQueryType is returned when a query is created with an empty type.
var ErrEmptyQueryType = errorfamily.NewRejection(
	"query.empty_query_type",
	"query type is required (got empty)",
)
