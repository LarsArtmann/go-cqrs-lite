package memory

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type MemoryStore struct {
	dispatcher.LifecycleMixin

	mu     sync.RWMutex
	events map[string][]event.Event
}

var _ event.Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	//nolint:exhaustruct // embedded LifecycleMixin has unexported fields from different package
	return &MemoryStore{
		events: make(map[string][]event.Event),
	}
}

func (s *MemoryStore) Save(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return errors.Wrap(err, "memory store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	existing := s.events[key]

	if len(existing) != expectedVersion.Int() {
		return errors.Wrapf(event.ErrVersionConflict,
			"expected version %d, got %d", expectedVersion, len(existing))
	}

	s.events[key] = append(existing, events...)

	return nil
}

func (s *MemoryStore) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return errors.Wrap(err, "memory store append batch")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	s.events[key] = append(s.events[key], events...)

	return nil
}

func (s *MemoryStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, errors.Wrap(err, "memory store load")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	result := make([]event.Event, len(events))
	copy(result, events)

	return result, nil
}

func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, errors.Wrap(err, "memory store load from version")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	if version.Int() >= len(events) {
		return []event.Event{}, nil
	}

	sub := events[version.Int():]
	result := make([]event.Event, len(sub))
	copy(result, sub)

	return result, nil
}

func (s *MemoryStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return errors.Wrap(err, "memory store delete")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	delete(s.events, key)

	return nil
}

func (s *MemoryStore) Close() error {
	return s.LifecycleMixin.Close() //nolint:wrapcheck
}
