package memory

import (
	"context"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// MemoryQueryStore is an in-memory implementation of query.QueryStore.
// It logs every query for audit purposes — "who queried what data and when?".
type MemoryQueryStore struct {
	dispatcher.Lifecycle

	mu      sync.RWMutex
	queries []*query.PersistedQuery
	idIndex map[id.RequestID]int
}

var (
	_ query.QueryStore           = (*MemoryQueryStore)(nil)
	_ query.QueryJournal         = (*MemoryQueryStore)(nil)
	_ query.SeekableQueryJournal = (*MemoryQueryStore)(nil)
	_ io.Closer                  = (*MemoryQueryStore)(nil)
)

func NewMemoryQueryStore() *MemoryQueryStore {
	return &MemoryQueryStore{
		idIndex: make(map[id.RequestID]int),
	}
}

// withWriteLock centralises the wrapClosed + Lock + defer Unlock preamble for
// write methods. The closure body runs under the lock.
func (s *MemoryQueryStore) withWriteLock(code, msg string, fn func() error) error {
	if err := wrapClosed(s.CheckClosed(query.ErrStoreClosed), code, msg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn()
}

// withQueryReadLock is the read-side companion to withWriteLock. It is a
// top-level generic function because Go does not permit generic methods; the
// [T] type parameter carries the read method's return type through the closure.
func withQueryReadLock[T any](
	s *MemoryQueryStore,
	code, msg string,
	fn func() (T, error),
) (T, error) {
	if err := wrapClosed(s.CheckClosed(query.ErrStoreClosed), code, msg); err != nil {
		var zero T

		return zero, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn()
}

func (s *MemoryQueryStore) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
	return s.withWriteLock("memory.save_query_failed", "memory query store save", func() error {
		s.idIndex[q.ID()] = len(s.queries)
		s.queries = append(s.queries, q)

		return nil
	})
}

func (s *MemoryQueryStore) LoadQueries(
	_ context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	return withQueryReadLock(s, "memory.load_queries_failed", "memory query store load queries", func() ([]*query.PersistedQuery, error) {
		result := make([]*query.PersistedQuery, 0, len(s.queries))

		for _, q := range s.queries {
			if q.ReceivedAt().After(after) {
				result = append(result, q)
			}
		}

		return result, nil
	})
}

func (s *MemoryQueryStore) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	return withQueryReadLock(s, "memory.readall_queries_failed", "memory query journal readall", func() ([]*query.PersistedQuery, error) {
		return slices.Clone(s.queries), nil
	})
}

func (s *MemoryQueryStore) ReadQueriesFrom(
	_ context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	return withQueryReadLock(s, "memory.readqueries_from_failed", "memory query journal readfrom", func() ([]*query.PersistedQuery, error) {
		startIdx := 0

		if afterRequestID != (id.RequestID{}) {
			idx, exists := s.idIndex[afterRequestID]
			if !exists {
				return nil, nil
			}

			startIdx = idx + 1
		}

		end := min(startIdx+limit, len(s.queries))

		if startIdx >= len(s.queries) {
			return nil, nil
		}

		return slices.Clone(s.queries[startIdx:end]), nil
	})
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryQueryStore) Close() error {
	return s.Lifecycle.Close()
}
