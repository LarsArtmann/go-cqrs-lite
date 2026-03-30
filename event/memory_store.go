package event

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
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
		mu:     sync.RWMutex{},
		events: make(map[string][]Event),
		closed: false,
	}
}

func (s *MemoryStore) streamKey(aggregateType AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

// Save appends events to the aggregate's event stream
func (s *MemoryStore) Save(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
	events []Event,
	expectedVersion Version,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	key := s.streamKey(aggregateType, aggregateID)
	existing := s.events[key]

	if len(existing) != expectedVersion.Int() {
		return errors.Wrapf(ErrVersionConflict,
			"expected version %d, got %d", expectedVersion, len(existing))
	}

	s.events[key] = append(existing, events...)
	return nil
}

// Load retrieves all events for an aggregate
func (s *MemoryStore) Load(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
) ([]Event, error) {
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

	return events[Version(0).Int():], nil
}

// LoadFromVersion retrieves events starting from a specific version
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
	version Version,
) ([]Event, error) {
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

	if version.Int() >= len(events) {
		return []Event{}, nil
	}

	return events[version.Int():], nil
}

// Delete removes all events for an aggregate
func (s *MemoryStore) Delete(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
) error {
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
