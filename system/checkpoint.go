package system

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// memoryCheckpointStore is a minimal in-memory checkpoint store for the
// projection host. It is used when no persistent checkpoint store is configured.
type memoryCheckpointStore struct {
	mu   sync.RWMutex
	data map[string]event.Checkpoint
}

func (s *memoryCheckpointStore) Save(
	_ context.Context,
	projection string,
	cp event.Checkpoint,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string]event.Checkpoint)
	}

	s.data[projection] = cp

	return nil
}

func (s *memoryCheckpointStore) Load(
	_ context.Context,
	projection string,
) (event.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data[projection], nil
}

func (s *memoryCheckpointStore) Close() error { return nil }

// Compile-time assertion: memoryCheckpointStore implements event.CheckpointStore.
var _ event.CheckpointStore = (*memoryCheckpointStore)(nil)
