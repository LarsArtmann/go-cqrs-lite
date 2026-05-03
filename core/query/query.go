package query

import "fmt"

// Type identifies a query type.
type Type string

// String returns the query type as a string.
func (t Type) String() string { return string(t) }

// Query represents a read-side query.
type Query interface {
	Type() Type
}

// Core provides a default implementation.
type Core struct {
	queryType Type
}

var _ Query = (*Core)(nil)

// Type returns the query type.
func (q *Core) Type() Type { return q.queryType }

// New creates a new query with validation.
func New(queryType Type) (*Core, error) {
	if queryType == "" {
		return nil, ErrEmptyQueryType
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
type Middleware func(Handler) Handler
