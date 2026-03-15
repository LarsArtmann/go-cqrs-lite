package event

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// MemoryStore is an in-memory implementation of Store for testing and development
type MemoryStore struct {
	mu     sync.RWMutex
	events map[string][]Event
	closed bool
}

// NewMemoryStore creates a new in-memory event store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: make(map[string][]Event),
	}
}

func (s *MemoryStore) streamKey(aggregateType AggregateType, aggregateID string) string {
	return string(aggregateType) + ":" + aggregateID
}

// Save appends events to the aggregate's event stream
func (s *MemoryStore) Save(ctx context.Context, aggregateType AggregateType, aggregateID string, events []Event, expectedVersion int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	key := s.streamKey(aggregateType, aggregateID)
	existing := s.events[key]

	if len(existing) != expectedVersion {
		return errors.Wrapf(ErrVersionConflict,
			"expected version %d, got %d", expectedVersion, len(existing))
	}

	s.events[key] = append(existing, events...)
	return nil
}

// Load retrieves all events for an aggregate
func (s *MemoryStore) Load(ctx context.Context, aggregateType AggregateType, aggregateID string) ([]Event, error) {
	return s.LoadFromVersion(ctx, aggregateType, aggregateID, 0)
}

// LoadFromVersion retrieves events starting from a specific version
func (s *MemoryStore) LoadFromVersion(ctx context.Context, aggregateType AggregateType, aggregateID string, version int) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrStoreClosed
	}

	key := s.streamKey(aggregateType, aggregateID)
	events, exists := s.events[key]
	if !exists {
		return nil, ErrAggregateNotFound
	}

	if version >= len(events) {
		return []Event{}, nil
	}

	return events[version:], nil
}

// Delete removes all events for an aggregate
func (s *MemoryStore) Delete(ctx context.Context, aggregateType AggregateType, aggregateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	key := s.streamKey(aggregateType, aggregateID)
	delete(s.events, key)
	return nil
}

// Close marks the store as closed
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
