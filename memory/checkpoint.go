package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
		return event.Checkpoint{}, fmt.Errorf(
			"load checkpoint for projection %q: %w",
			projectionName,
			err,
		)
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
		return fmt.Errorf("save checkpoint for projection %q: %w", projectionName, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[projectionName] = checkpoint

	return nil
}

// Close marks the store as closed.
func (s *MemoryCheckpointStore) Close() error {
	return s.Lifecycle.Close() //nolint:wrapcheck // transparent delegation, caller wraps
}

var (
	_ event.CheckpointSink   = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointSource = (*MemoryCheckpointStore)(nil)
	_ event.CheckpointStore  = (*MemoryCheckpointStore)(nil)
)
