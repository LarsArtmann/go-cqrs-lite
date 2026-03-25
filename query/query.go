package query

// Type identifies a query type
type Type string

// Query represents a read-side query
type Query interface {
	Type() Type
}

// BaseQuery provides a default implementation
type BaseQuery struct {
	queryType Type
}

func (q *BaseQuery) Type() Type { return q.queryType }

// New creates a new query
func New(queryType Type) *BaseQuery {
	return &BaseQuery{queryType: queryType}
}

// Result contains query results
type Result[T any] struct {
	Data  T
	Error error
}

// QueryHandler processes a query and returns a result
type QueryHandler[T any] func(query Query) (T, error)

// Middleware wraps query handlers for cross-cutting concerns
type Middleware func(func(Query) (any, error)) func(Query) (any, error)
