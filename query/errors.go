package query

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrHandlerNotFound is returned when no handler is registered for a query type.
var ErrHandlerNotFound error = errorfamily.NewRejection(
	"query.handler_not_found",
	"no handler registered for query",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed error = errorfamily.NewInfrastructure(
	"query.dispatcher_closed",
	"query dispatcher is closed",
)

// ErrEmptyQueryType is returned when a query is created with an empty type.
var ErrEmptyQueryType error = errorfamily.NewRejection(
	"query.empty_query_type",
	"query type is required (got empty)",
)

// ErrTypeAssertion is returned when a query cannot be type-asserted to the expected type.
var ErrTypeAssertion error = errorfamily.NewRejection(
	"query.type_assertion",
	"query type assertion failed",
)

// ErrStoreClosed is returned when the query store is closed.
var ErrStoreClosed error = errorfamily.NewInfrastructure(
	"query.store_closed",
	"query store is closed",
)

// ErrQueryNotFound is returned when a query is not found in the store.
var ErrQueryNotFound error = errorfamily.NewRejection(
	"query.not_found",
	"query not found",
)

// ErrDuplicateQuery is returned when a query with the same ID already exists.
var ErrDuplicateQuery error = errorfamily.NewConflict(
	"query.duplicate",
	"query with this ID already exists",
)
