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
	return func(a *QueryAdapter) { a.Serialize = true }
}

// QueryAdapter wraps a [metaengine.StreamLogBackend] as a [query.QueryStore].
// Queries are stream-keyed append-only logs, just like events and commands.
type QueryAdapter struct {
	AdapterCore[*query.PersistedQuery]
}

// NewQueryAdapter creates a query.QueryStore backed by a StreamLogBackend.
func NewQueryAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...QueryAdapterOption,
) *QueryAdapter {
	a := &QueryAdapter{}
	a.AdapterCore = AdapterCore[*query.PersistedQuery]{
		Backend:    backend,
		Collection: collection,
		Noun:       "query",
		Encode:     a.encodeQuery,
		Decode:     a.decodeQuery,
		IDOf:       func(q *query.PersistedQuery) string { return q.ID().String() },
	}

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

	return a.Backend.StreamAppend(
		ctx,
		a.Collection,
		sid,
		a.ToAny([]*query.PersistedQuery{q}),
	)
}

func (a *QueryAdapter) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	values, err := a.Backend.JournalReadAll(ctx, a.Collection)
	if err != nil {
		return nil, fmt.Errorf("query adapter: load queries: %w", err)
	}

	all, err := a.FromAny(values)
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

// ReadAllQueries is promoted ReadAll renamed to satisfy query.QueryJournal.

func (a *QueryAdapter) ReadAllQueries(ctx context.Context) ([]*query.PersistedQuery, error) {
	return a.ReadAll(ctx)
}

func (a *QueryAdapter) ReadQueriesFrom(
	ctx context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	after := ""
	if afterRequestID != (id.RequestID{}) {
		after = afterRequestID.String()
	}

	return a.ReadFromAfter(ctx, after, limit)
}

func (a *QueryAdapter) Close() error { return nil }
