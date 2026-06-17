package query

import (
	"context"

	ro "github.com/samber/ro"
)

// QueryBus is a reactive subject for query streams.
// Use NewQueryBus() to create one. Subscribe with ro.Observer, emit with Next.
//
// This mirrors event.EventBus for query-side reactive dispatch. Queries
// published via ro.Subject.Next(q) are broadcast to all subscribers.
//
// Example:
//
//	bus := query.NewQueryBus()
//	filtered := ro.Pipe1(bus, query.FilterQueryType("user.get"))
//	filtered.Subscribe(query.HandlerToObserver(myHandler))
//	bus.Next(getQuery)
type QueryBus = ro.Subject[Query]

// NewQueryBus creates a new PublishSubject-backed QueryBus for
// broadcasting queries to multiple subscribers.
func NewQueryBus() ro.Subject[Query] {
	return ro.NewPublishSubject[Query]()
}

// NewReplayQueryBus creates a new ReplaySubject-backed QueryBus that
// replays the last n queries to new subscribers.
func NewReplayQueryBus(n int) ro.Subject[Query] {
	return ro.NewReplaySubject[Query](n)
}

// NewBehaviorQueryBus creates a new BehaviorSubject-backed QueryBus that
// replays the latest query to new subscribers.
func NewBehaviorQueryBus(initial Query) ro.Subject[Query] {
	return ro.NewBehaviorSubject(initial)
}

// FilterQueryType returns an operator that filters an Observable[Query]
// to only queries of the given type.
func FilterQueryType(qType Type) func(ro.Observable[Query]) ro.Observable[Query] {
	return ro.Filter(func(q Query) bool {
		return q.Type() == qType
	})
}

// FilterQueryTypes returns an operator that filters an Observable[Query]
// to only queries of the given types.
func FilterQueryTypes(qTypes ...Type) func(ro.Observable[Query]) ro.Observable[Query] {
	types := newQueryTypeSet(qTypes)

	return ro.Filter(func(q Query) bool {
		return types.has(q.Type())
	})
}

// HandlerToObserver converts a query Handler into a ro.Observer[Query].
// The handler receives the context from the stream (via NextWithContext/SubscribeWithContext).
// If the handler returns an error, the error is forwarded through the observer's error channel.
func HandlerToObserver(handler Handler) ro.Observer[Query] {
	var obs ro.Observer[Query]

	obs = ro.NewObserverWithContext(
		func(ctx context.Context, q Query) {
			_, err := handler(ctx, q)
			if err != nil {
				obs.ErrorWithContext(ctx, err)
			}
		},
		func(_ context.Context, _ error) {},
		func(_ context.Context) {},
	)

	return obs
}

// Observable is a named type for query observables, improving discoverability
// over the raw ro.Observable[Query].
type Observable = ro.Observable[Query]

type queryTypeSet map[Type]struct{}

func newQueryTypeSet(types []Type) queryTypeSet {
	if len(types) == 0 {
		return nil
	}

	s := make(queryTypeSet, len(types))
	for _, t := range types {
		s[t] = struct{}{}
	}

	return s
}

func (s queryTypeSet) has(t Type) bool {
	if s == nil {
		return true
	}

	_, ok := s[t]

	return ok
}
