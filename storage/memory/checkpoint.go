package memory

import (
	"context"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
	err := s.CheckClosed(dispatcher.ErrDispatcherClosed)
	if err != nil {
		return event.Checkpoint{}, errorfamily.Wrapf(err, errorfamily.Infrastructure,
			"memory.checkpoint_load", "load checkpoint for projection %q", projectionName)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.checkpoints[projectionName], nil
}

// Save persists the checkpoint for a projection.
func (s *MemoryCheckpointStore) Save(
	_ context.Context,
	projectionName string,
	checkpoint event.Checkpoint,
) error {
	err := s.CheckClosed(dispatcher.ErrDispatcherClosed)
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure,
			"memory.checkpoint_save", "save checkpoint for projection %q", projectionName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[projectionName] = checkpoint

	return nil
}

// Close marks the store as closed.
func (s *MemoryCheckpointStore) Close() error {
	return s.Lifecycle.Close()
}

var (
	_ event.CheckpointSink   = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointSource = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointStore  = (*MemoryCheckpointStore)(nil)
)
