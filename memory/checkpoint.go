package memory

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// MemoryCheckpointStore is an in-memory CheckpointStore for testing.
type MemoryCheckpointStore struct {
	mu          sync.RWMutex
	checkpoints map[string]id.EventID
}

// NewMemoryCheckpointStore creates a new empty MemoryCheckpointStore.
func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{
		checkpoints: make(map[string]id.EventID),
		mu:          sync.RWMutex{},
	}
}

// Load returns the last processed event ID for a projection.
func (s *MemoryCheckpointStore) Load(_ context.Context, projectionName string) (id.EventID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.checkpoints[projectionName], nil
}

// Save persists the checkpoint for a projection.
func (s *MemoryCheckpointStore) Save(
	_ context.Context,
	projectionName string,
	eventID id.EventID,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[projectionName] = eventID

	return nil
}

// Close is a no-op for the in-memory checkpoint store.
func (s *MemoryCheckpointStore) Close() error { return nil }

var _ event.CheckpointStore = (*MemoryCheckpointStore)(nil)
