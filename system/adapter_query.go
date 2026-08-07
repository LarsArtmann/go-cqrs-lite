package system

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// QueryAdapterOption tunes a QueryAdapter at construction time.
type QueryAdapterOption func(*QueryAdapter)

// WithQuerySerialization enables query serialization for persistent engines
// (SQLite, Pebble). When enabled, queries are encoded to JSON envelope strings
// on write and decoded on read. For the Memory engine, this option should NOT
// be set — queries are stored as direct pointers.
func WithQuerySerialization() QueryAdapterOption {
	return func(a *QueryAdapter) { a.serialize = true }
}

// QueryAdapter wraps a [metaengine.StreamLogBackend] as a [query.QueryStore].
// Queries are stream-keyed append-only logs, just like events and commands.
type QueryAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
	serialize  bool
}

// NewQueryAdapter creates a query.QueryStore backed by a StreamLogBackend.
func NewQueryAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...QueryAdapterOption,
) *QueryAdapter {
	a := &QueryAdapter{backend: backend, collection: collection}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Compile-time assertions.
var (
	_ query.QueryStore           = (*QueryAdapter)(nil)
	_ query.SeekableQueryJournal = (*QueryAdapter)(nil)
)

func (a *QueryAdapter) SaveQuery(ctx context.Context, q *query.PersistedQuery) error {
	// Use the query's request ID as the stream key for per-query isolation.
	sid := q.ID().String()

	return a.backend.StreamAppend(
		ctx,
		a.collection,
		sid,
		a.queriesToAny([]*query.PersistedQuery{q}),
	)
}

func (a *QueryAdapter) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("query adapter: load queries: %w", err)
	}

	all, err := a.anyToQueries(values)
	if err != nil {
		return nil, fmt.Errorf("query adapter: load queries: %w", err)
	}

	result := make([]*query.PersistedQuery, 0, len(all))
	for _, q := range all {
		if q.ReceivedAt().After(after) {
			result = append(result, q)
		}
	}

	return result, nil
}

func (a *QueryAdapter) ReadAllQueries(ctx context.Context) ([]*query.PersistedQuery, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("query adapter: read all: %w", err)
	}

	return a.anyToQueries(values)
}

func (a *QueryAdapter) ReadQueriesFrom(
	ctx context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	afterSeq := int64(0)

	if afterRequestID != (id.RequestID{}) {
		all, err := a.backend.JournalReadAll(ctx, a.collection)
		if err != nil {
			return nil, fmt.Errorf("query adapter: read from: %w", err)
		}

		queries, err := a.anyToQueries(all)
		if err != nil {
			return nil, fmt.Errorf("query adapter: read from: %w", err)
		}

		for i, q := range queries {
			if q.ID() == afterRequestID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("query adapter: read from: %w", err)
	}

	return a.anyToQueries(values)
}

func (a *QueryAdapter) Close() error { return nil }
