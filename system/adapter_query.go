package system

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// QueryAdapter wraps a [metaengine.StreamLogBackend] as a [query.QueryStore].
// Queries are stream-keyed append-only logs, just like events and commands.
type QueryAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
}

// NewQueryAdapter creates a query.QueryStore backed by a StreamLogBackend.
func NewQueryAdapter(backend metaengine.StreamLogBackend, collection string) *QueryAdapter {
	return &QueryAdapter{backend: backend, collection: collection}
}

// Compile-time assertions.
var (
	_ query.QueryStore           = (*QueryAdapter)(nil)
	_ query.SeekableQueryJournal = (*QueryAdapter)(nil)
)

func (a *QueryAdapter) SaveQuery(ctx context.Context, q *query.PersistedQuery) error {
	// Use the query's request ID as the stream key for per-query isolation.
	sid := q.ID().String()

	return a.backend.StreamAppend(ctx, a.collection, sid, []any{q})
}

func (a *QueryAdapter) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("query adapter: load queries: %w", err)
	}

	result := make([]*query.PersistedQuery, 0, len(values))
	for _, val := range values {
		q, ok := val.(*query.PersistedQuery)
		if !ok {
			continue
		}

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

	result := make([]*query.PersistedQuery, 0, len(values))
	for _, val := range values {
		q, ok := val.(*query.PersistedQuery)
		if !ok {
			continue
		}

		result = append(result, q)
	}

	return result, nil
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

		for i, val := range all {
			q, ok := val.(*query.PersistedQuery)
			if ok && q.ID() == afterRequestID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("query adapter: read from: %w", err)
	}

	result := make([]*query.PersistedQuery, 0, len(values))
	for _, val := range values {
		q, ok := val.(*query.PersistedQuery)
		if !ok {
			continue
		}

		result = append(result, q)
	}

	return result, nil
}

func (a *QueryAdapter) Close() error { return nil }
