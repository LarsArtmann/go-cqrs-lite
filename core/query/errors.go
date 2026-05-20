package query

import "github.com/larsartmann/go-cqrs-lite/core/event"

// ErrQueryNotSupported is returned when a query type is not supported.
var ErrQueryNotSupported = event.NewRejection(
	"query.not_supported",
	"query not supported",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = event.NewInfrastructure(
	"query.dispatcher_closed",
	"query dispatcher is closed",
)

// ErrEmptyQueryType is returned when a query is created with an empty type.
var ErrEmptyQueryType = event.NewRejection(
	"query.empty_query_type",
	"query type is required (got empty)",
)
