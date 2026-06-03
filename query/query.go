package query

import (
	"context"
	"fmt"
)

// Type identifies a query type.
type Type string

// String returns the query type as a string.
func (t Type) String() string { return string(t) }

// Query represents a read-side query.
type Query interface {
	Type() Type
}

// BasicQuery provides a default implementation.
type BasicQuery struct {
	queryType Type
}

var _ Query = (*BasicQuery)(nil)

// Type returns the query type.
func (q *BasicQuery) Type() Type { return q.queryType }

// New creates a new query with validation.
func New(queryType Type) (*BasicQuery, error) {
	if queryType == "" {
		return nil, ErrEmptyQueryType
	}

	return &BasicQuery{queryType: queryType}, nil
}

// MustNew creates a new query or panics on validation failure.
// Use only in tests where inputs are guaranteed valid.
func MustNew(queryType Type) *BasicQuery {
	q, err := New(queryType)
	if err != nil {
		panic(fmt.Sprintf("query.MustNew: %v", err))
	}

	return q
}

// Middleware wraps query handlers for cross-cutting concerns.
type Middleware func(Handler) Handler

// TypedHandler processes a typed query and returns a typed result.
// Q is the concrete query type, R is the result type.
// Use with RegisterTyped for compile-time type safety at registration,
// eliminating the need for manual type assertions in handlers.
type TypedHandler[Q Query, R any] func(ctx context.Context, q Q) (R, error)
