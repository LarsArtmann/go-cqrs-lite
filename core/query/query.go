package query

import "context"

// Type identifies a query type.
type Type string

// Query represents a read-side query.
type Query interface {
	Type() Type
}

// Core provides a default implementation.
type Core struct {
	queryType Type
}

// Type returns the query type.
func (q *Core) Type() Type { return q.queryType }

// New creates a new query.
func New(queryType Type) *Core {
	return &Core{queryType: queryType}
}

// Middleware wraps query handlers for cross-cutting concerns.
type Middleware func(func(context.Context, Query) (any, error)) func(context.Context, Query) (any, error)
