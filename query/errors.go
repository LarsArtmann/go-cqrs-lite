package query

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrHandlerNotFound is returned when no handler is registered for a query type.
var ErrHandlerNotFound = event.NewRejection(
	"query.handler_not_found",
	"no handler registered for query",
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

// ErrTypeAssertion is returned when a query cannot be type-asserted to the expected type.
var ErrTypeAssertion = event.NewRejection(
	"query.type_assertion",
	"query type assertion failed",
)

// ErrStoreClosed is returned when the query store is closed.
var ErrStoreClosed = event.NewInfrastructure(
	"query.store_closed",
	"query store is closed",
)

// ErrQueryNotFound is returned when a query is not found in the store.
var ErrQueryNotFound = event.NewRejection(
	"query.not_found",
	"query not found",
)

// ErrDuplicateQuery is returned when a query with the same ID already exists.
var ErrDuplicateQuery = event.NewConflict(
	"query.duplicate",
	"query with this ID already exists",
)
