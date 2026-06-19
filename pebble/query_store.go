package pebble

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// QueryStore implements query.QueryStore, query.QueryJournal, and
// query.SeekableQueryJournal backed by Pebble.
//
// Queries are global audit records ("who queried what data and when?"). Unlike
// commands, queries have no per-aggregate index — they are stored in a single
// journal key space:
//
//   - cqrs_query:{requestID} — global journal, ordered by request ID
//
// RequestIDs are ULIDs (time-sortable), so journal keys are naturally ordered
// by receipt time.
//
// The store shares the Pebble DB with other stores via disjoint key prefixes.
type QueryStore struct {
	storeBase
}

// QueryStoreOption configures a QueryStore.
type QueryStoreOption func(*QueryStore)

// WithQueryAsyncWrites disables sync writes for higher throughput at the cost
// of durability guarantees.
func WithQueryAsyncWrites() QueryStoreOption {
	return func(s *QueryStore) { s.syncWrites = false }
}

// WithQueryPrefix overrides the default key prefix ("cqrs_query:").
func WithQueryPrefix(p string) QueryStoreOption {
	return func(s *QueryStore) { s.prefix = p }
}

// NewQueryStore creates a new QueryStore using an existing Pebble DB.
// Panics if db is nil.
func NewQueryStore(
	database *pebble.DB,
	logger *slog.Logger,
	opts ...QueryStoreOption,
) *QueryStore {
	if database == nil {
		panic("pebble: NewQueryStore called with nil db")
	}

	s := &QueryStore{
		storeBase: storeBase{
			db:         database,
			logger:     logger,
			prefix:     "cqrs_query:",
			syncWrites: true,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Close is a no-op; the underlying *pebble.DB is owned by the caller (or Backend).
func (s *QueryStore) Close() error { return nil }

// SaveQuery persists a single query with duplicate-ID detection.
// Returns query.ErrDuplicateQuery if the request ID already exists.
func (s *QueryStore) SaveQuery(
	ctx context.Context,
	q *query.PersistedQuery,
) error {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.query.save",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrString("query.type", string(q.Type()))))
	defer span.End()

	key := s.queryKey(q.ID())

	if s.queryExists(key) {
		return query.WrapConflict(query.ErrDuplicateQuery, "pebble.duplicate_query",
			fmt.Sprintf("query %s already exists", q.ID()))
	}

	data, err := s.serializeQuery(q)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return query.WrapCorruption(err, "pebble.serialize_query",
			fmt.Sprintf("serialize query %s", q.ID()))
	}

	if err := s.db.Set(key, data, s.writeOptions()); err != nil {
		cqrsotel.RecordError(span, err)

		return query.WrapInfrastructure(err, "pebble.query_write",
			fmt.Sprintf("write query %s", q.ID()))
	}

	return nil
}

func (s *QueryStore) queryExists(key []byte) bool {
	_, closer, err := s.db.Get(key)
	if err == nil {
		_ = closer.Close()

		return true
	}

	return !errors.Is(err, pebble.ErrNotFound)
}

// ── Key generation ──────────────────────────────────────────────────────────

func (s *QueryStore) queryKey(requestID id.RequestID) []byte {
	return fmt.Appendf(nil, "%s%s", s.prefix, requestID)
}

// Ensure QueryStore implements query.QueryStore, journal, and seekable journal.
var (
	_ query.QueryStore           = (*QueryStore)(nil)
	_ query.QueryJournal         = (*QueryStore)(nil)
	_ query.SeekableQueryJournal = (*QueryStore)(nil)
	_ io.Closer                  = (*QueryStore)(nil)
)
