package memory

import (
	"context"
	"fmt"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// MemoryQueryStore is an in-memory implementation of query.QueryStore.
// It logs every query for audit purposes — "who queried what data and when?".
//
// It embeds the generic [LogStore] core. Queries have no per-stream
// scoping, so stream tracking is disabled; the store is a single global log
// keyed by request ID.
type MemoryQueryStore struct {
*LogStore[*query.PersistedQuery, id.RequestID]
}

var (
	_ query.QueryStore           = (*MemoryQueryStore)(nil)
	_ query.QueryJournal         = (*MemoryQueryStore)(nil)
	_ query.SeekableQueryJournal = (*MemoryQueryStore)(nil)
	_ io.Closer                  = (*MemoryQueryStore)(nil)
)

func NewMemoryQueryStore() *MemoryQueryStore {
	return &MemoryQueryStore{
		LogStore: NewLogStore(LogStoreConfig[*query.PersistedQuery, id.RequestID]{
			GetID:     func(q *query.PersistedQuery) id.RequestID { return q.ID() },
			IsZeroID:  func(requestID id.RequestID) bool { return requestID == (id.RequestID{}) },
			ClosedErr: query.ErrStoreClosed,
			NewDupErr: func(requestID id.RequestID, _ string) error {
				return errorfamily.WrapConflict(
					query.ErrDuplicateQuery,
					"memory.duplicate_query",
					fmt.Sprintf("query with ID %s already exists", requestID))
			},
			NewNotFound:  nil,
			TrackStreams: false,
		}),
	}
}

func (s *MemoryQueryStore) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
	return s.WithWrite("memory.save_query_failed", "memory query store save", func() error {
		if dupErr := s.CheckDuplicateLocked(q.ID(), ""); dupErr != nil {
			return dupErr
		}

		s.AppendLocked("", []*query.PersistedQuery{q})

		return nil
	})
}

func (s *MemoryQueryStore) LoadQueries(
	_ context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	return WithReadLock(
		s.LogStore,
		"memory.load_queries_failed",
		"memory query store load queries",
		func() ([]*query.PersistedQuery, error) {
			return s.FilterAllLocked(func(q *query.PersistedQuery) bool {
				return q.ReceivedAt().After(after)
			}), nil
		},
	)
}

func (s *MemoryQueryStore) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	return WithReadLock(
		s.LogStore,
		"memory.readall_queries_failed",
		"memory query journal readall",
		func() ([]*query.PersistedQuery, error) {
			return s.ReadAllLocked(), nil
		},
	)
}

// ReadQueriesFrom returns queries after the given request ID, ordered by
// insertion. A missing start position returns nothing — an unknown position
// means nothing new.
func (s *MemoryQueryStore) ReadQueriesFrom(
	_ context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	return WithReadLock(
		s.LogStore,
		"memory.readqueries_from_failed",
		"memory query journal readfrom",
		func() ([]*query.PersistedQuery, error) {
			return s.ReadFromLocked(afterRequestID, limit, false), nil
		},
	)
}
