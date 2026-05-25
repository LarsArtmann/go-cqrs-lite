package query

import errorfamily "github.com/larsartmann/go-error-family"

// ErrQueryNotSupported is returned when a query type is not supported.
var ErrQueryNotSupported = errorfamily.NewRejection(
	"query.not_supported",
	"query not supported",
)

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
