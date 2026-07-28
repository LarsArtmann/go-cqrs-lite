package memory

import (
	"context"
	"sync"

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
	return withCheckpointReadLock(
		s,
		"memory.checkpoint_load",
		"checkpoint store load",
		func() (event.Checkpoint, error) {
			return s.checkpoints[projectionName], nil
		},
	)
}

// Save persists the checkpoint for a projection.
func (s *MemoryCheckpointStore) Save(
	_ context.Context,
	projectionName string,
	checkpoint event.Checkpoint,
) error {
	return s.withWriteLock("memory.checkpoint_save", "checkpoint store save", func() error {
		s.checkpoints[projectionName] = checkpoint

		return nil
	})
}

// withWriteLock checks the store is open, acquires the write lock, and runs fn
// under the lock.
func (s *MemoryCheckpointStore) withWriteLock(code, msg string, fn func() error) error {
	if err := wrapClosed(s.CheckClosed(dispatcher.ErrDispatcherClosed), code, msg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn()
}

// withCheckpointReadLock is the read-side companion to withWriteLock.
func withCheckpointReadLock[T any](
	s *MemoryCheckpointStore,
	code, msg string,
	fn func() (T, error),
) (T, error) {
	if err := wrapClosed(s.CheckClosed(dispatcher.ErrDispatcherClosed), code, msg); err != nil {
		var zero T

		return zero, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn()
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
