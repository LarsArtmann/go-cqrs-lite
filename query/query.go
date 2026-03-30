package query

// Type identifies a query type
type Type string

// Query represents a read-side query
type Query interface {
	Type() Type
}

// Core provides a default implementation
type Core struct {
	queryType Type
}

// Type returns the query type.
func (q *Core) Type() Type { return q.queryType }

// New creates a new query
func New(queryType Type) *Core {
	return &Core{queryType: queryType}
}

// Result contains query results
type Result[T any] struct {
	Data  T
	Error error
}

// Middleware wraps query handlers for cross-cutting concerns
type Middleware func(func(Query) (any, error)) func(Query) (any, error)
