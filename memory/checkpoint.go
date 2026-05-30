package memory

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// MemoryCheckpointStore is an in-memory CheckpointStore for testing.
type MemoryCheckpointStore struct {
	dispatcher.Lifecycle

	mu          sync.RWMutex
	checkpoints map[string]event.Checkpoint
}

// NewMemoryCheckpointStore creates a new empty MemoryCheckpointStore.
func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{
		Lifecycle:   dispatcher.Lifecycle{},
		checkpoints: make(map[string]event.Checkpoint),
		mu:          sync.RWMutex{},
	}
}

// Load returns the last processed event ID for a projection.
func (s *MemoryCheckpointStore) Load(
	_ context.Context,
	projectionName string,
) (event.Checkpoint, error) {
	if err := s.CheckClosed(dispatcher.ErrDispatcherClosed); err != nil {
		return event.Checkpoint{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.checkpoints[projectionName], nil
}

// Save persists the checkpoint for a projection.
func (s *MemoryCheckpointStore) Save(
	_ context.Context,
	projectionName string,
	cp event.Checkpoint,
) error {
	if err := s.CheckClosed(dispatcher.ErrDispatcherClosed); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[projectionName] = cp

	return nil
}

// Close marks the store as closed.
func (s *MemoryCheckpointStore) Close() error {
	return s.Lifecycle.Close() //nolint:wrapcheck
}

var (
	_ event.CheckpointSink   = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointSource = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointStore  = (*MemoryCheckpointStore)(nil)
)
