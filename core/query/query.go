package query

import (
	"context"
	"fmt"
)

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

// New creates a new query with validation.
func New(queryType Type) (*Core, error) {
	if queryType == "" {
		//nolint:err113 // dynamic error required for validation
		return nil, fmt.Errorf("query type is required (got empty)")
	}

	return &Core{queryType: queryType}, nil
}

// MustNew creates a new query or panics on validation failure.
// Use only in tests where inputs are guaranteed valid.
func MustNew(queryType Type) *Core {
	q, err := New(queryType)
	if err != nil {
		panic(fmt.Sprintf("query.MustNew: %v", err))
	}

	return q
}

// Middleware wraps query handlers for cross-cutting concerns.
type Middleware func(func(context.Context, Query) (any, error)) func(context.Context, Query) (any, error)
