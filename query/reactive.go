package query

import (
	ro "github.com/samber/ro"
)

// QueryBus is a reactive subject for query streams.
// Use NewQueryBus() to create one. Subscribe with ro.Observer, emit with Next.
type QueryBus = ro.Subject[Query]

// NewQueryBus creates a new PublishSubject-backed QueryBus for broadcasting queries.
func NewQueryBus() ro.Subject[Query] {
	return ro.NewPublishSubject[Query]()
}

// FilterQueryType returns an operator that filters an Observable[Query] to only queries of the given type.
func FilterQueryType(queryType Type) func(ro.Observable[Query]) ro.Observable[Query] {
	return ro.Filter[Query](func(q Query) bool {
		return q.Type() == queryType
	})
}
