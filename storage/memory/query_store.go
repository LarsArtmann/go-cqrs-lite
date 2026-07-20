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

func (s *MemoryQueryStore) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
	if err := wrapClosed(
		s.CheckClosed(query.ErrStoreClosed),
		"memory.save_query_failed",
		"memory query store save",
	); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.idIndex[q.ID()] = len(s.queries)
	s.queries = append(s.queries, q)

	return nil
}

func (s *MemoryQueryStore) LoadQueries(
	_ context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	if err := wrapClosed(
		s.CheckClosed(query.ErrStoreClosed),
		"memory.load_queries_failed",
		"memory query store load queries",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*query.PersistedQuery, 0, len(s.queries))

	for _, q := range s.queries {
		if q.ReceivedAt().After(after) {
			result = append(result, q)
		}
	}

	return result, nil
}

func (s *MemoryQueryStore) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	if err := wrapClosed(
		s.CheckClosed(query.ErrStoreClosed),
		"memory.readall_queries_failed",
		"memory query journal readall",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.queries), nil
}

func (s *MemoryQueryStore) ReadQueriesFrom(
	_ context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	if err := wrapClosed(
		s.CheckClosed(query.ErrStoreClosed),
		"memory.readqueries_from_failed",
		"memory query journal readfrom",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

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
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryQueryStore) Close() error {
	return s.Lifecycle.Close()
}
