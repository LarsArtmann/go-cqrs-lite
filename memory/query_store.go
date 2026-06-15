package memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// MemoryQueryStore is an in-memory implementation of query.QueryStore.
// It logs every query for audit purposes — "who queried what data and when?".
type MemoryQueryStore struct {
	mu      sync.RWMutex
	queries []*query.PersistedQuery
	idIndex map[id.RequestID]int
}

var (
	_ query.QueryStore           = (*MemoryQueryStore)(nil)
	_ query.QueryJournal         = (*MemoryQueryStore)(nil)
	_ query.SeekableQueryJournal = (*MemoryQueryStore)(nil)
)

func NewMemoryQueryStore() *MemoryQueryStore {
	return &MemoryQueryStore{ //nolint:exhaustruct // zero-valued sync.RWMutex is ready
		idIndex: make(map[id.RequestID]int),
	}
}

func (s *MemoryQueryStore) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*query.PersistedQuery

	for _, q := range s.queries {
		if q.ReceivedAt().After(after) {
			result = append(result, q)
		}
	}

	return result, nil
}

func (s *MemoryQueryStore) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.queries), nil
}

func (s *MemoryQueryStore) ReadQueriesFrom(
	_ context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
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

func (s *MemoryQueryStore) Close() error { return nil }
