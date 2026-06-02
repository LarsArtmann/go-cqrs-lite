package memory

import (
	"context"
	"io"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// MemoryStore is an in-memory implementation of event.Store and event.Journal.
// It is safe for concurrent use. Designed for testing and single-process deployments.
type MemoryStore struct {
	dispatcher.Lifecycle

	mu     sync.RWMutex
	events map[string][]event.Event
}

var (
	_ event.Store           = (*MemoryStore)(nil)
	_ event.Journal         = (*MemoryStore)(nil)
	_ event.SeekableJournal = (*MemoryStore)(nil)
	_ event.BackwardsSource = (*MemoryStore)(nil)
	_ io.Closer             = (*MemoryStore)(nil)
)

// NewMemoryStore creates a new in-memory event store.
func NewMemoryStore() *MemoryStore {
	//nolint:exhaustruct // embedded Lifecycle has unexported fields from different package
	return &MemoryStore{
		events: make(map[string][]event.Event),
	}
}

// Save appends events to an aggregate stream with optimistic concurrency check.
// Returns ErrVersionConflict if the expected version does not match the current stream length.
func (s *MemoryStore) Save(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.save_failed", "memory store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	existing := s.events[key]

	err = event.CheckVersionConflict(len(existing), expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.save_failed", "memory store save")
	}

	s.events[key] = append(existing, events...)

	return nil
}

// AppendBatch appends events without a version check. Useful for testing idempotent writes.
func (s *MemoryStore) AppendBatch(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"memory.append_batch_failed",
			"memory store append batch",
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	s.events[key] = append(s.events[key], events...)

	return nil
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryStore) Close() error {
	return s.Lifecycle.Close() //nolint:wrapcheck
}
