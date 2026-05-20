package memory

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// MemoryStore is an in-memory implementation of event.Store and event.GlobalLoader.
// It is safe for concurrent use. Designed for testing and single-process deployments.
type MemoryStore struct {
	dispatcher.LifecycleMixin

	mu     sync.RWMutex
	events map[string][]event.Event
}

var (
	_ event.Store            = (*MemoryStore)(nil)
	_ event.GlobalLoader     = (*MemoryStore)(nil)
	_ event.PositionalLoader = (*MemoryStore)(nil)
	_ io.Closer              = (*MemoryStore)(nil)
)

// NewMemoryStore creates a new in-memory event store.
func NewMemoryStore() *MemoryStore {
	//nolint:exhaustruct // embedded LifecycleMixin has unexported fields from different package
	return &MemoryStore{
		events: make(map[string][]event.Event),
	}
}

// Save appends events to an aggregate stream with optimistic concurrency check.
// Returns ErrVersionConflict if the expected version does not match the current stream length.
func (s *MemoryStore) Save(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return fmt.Errorf("memory store save: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	existing := s.events[key]

	if len(existing) != expectedVersion.Int() {
		return fmt.Errorf(
			"%w: expected version %d, got %d",
			event.ErrVersionConflict,
			expectedVersion,
			len(existing),
		)
	}

	s.events[key] = append(existing, events...)

	return nil
}

// AppendBatch appends events without a version check. Useful for testing idempotent writes.
func (s *MemoryStore) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return fmt.Errorf("memory store append batch: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	s.events[key] = append(s.events[key], events...)

	return nil
}

// Delete removes all events for an aggregate.
func (s *MemoryStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return fmt.Errorf("memory store delete: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := streamKey(aggregateType, aggregateID)
	delete(s.events, key)

	return nil
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryStore) Close() error {
	return s.LifecycleMixin.Close() //nolint:wrapcheck
}
