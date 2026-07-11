package memory

import (
	"context"
	"io"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// MemoryStore is an in-memory implementation of event.Store and event.Journal.
// It is safe for concurrent use. Designed for testing and single-process deployments.
type MemoryStore struct {
	dispatcher.Lifecycle

	mu           sync.RWMutex
	globalLog    []event.Event      // canonical event storage (single copy)
	streamIndex  map[string][]int   // streamKey → indices into globalLog
	eventIDIndex map[id.EventID]int // index into globalLog for SeekableJournal
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
	return &MemoryStore{
		streamIndex:  make(map[string][]int),
		eventIDIndex: make(map[id.EventID]int),
	}
}

// Save appends events to an aggregate stream with optimistic concurrency check.
// Returns ErrVersionConflict if the expected version does not match the current stream length.
func (s *MemoryStore) Save(
	_ context.Context,
	ref id.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "memory.save_failed", "memory store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	streamLen := len(s.streamIndex[key])

	err = event.CheckVersionConflict(streamLen, expectedVersion)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "memory.save_failed", "memory store save")
	}

	s.appendToGlobalLog(key, events)

	return nil
}

// AppendBatch appends events without a version check. Useful for testing idempotent writes.
func (s *MemoryStore) AppendBatch(
	_ context.Context,
	ref id.AggregateRef,
	events []event.Event,
) error {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"memory.append_batch_failed",
			"memory store append batch",
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	s.appendToGlobalLog(key, events)

	return nil
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryStore) Close() error {
	return s.Lifecycle.Close()
}

func (s *MemoryStore) appendToGlobalLog(streamKey string, events []event.Event) {
	for _, evt := range events {
		idx := len(s.globalLog)
		s.eventIDIndex[evt.ID()] = idx
		s.globalLog = append(s.globalLog, evt)
		s.streamIndex[streamKey] = append(s.streamIndex[streamKey], idx)
	}
}
