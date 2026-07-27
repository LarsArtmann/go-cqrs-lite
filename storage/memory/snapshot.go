package memory

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	snappkg "github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

type MemorySnapshotStore struct {
	dispatcher.Lifecycle

	mu        sync.RWMutex
	snapshots map[string]*snappkg.Snapshot
}

var (
	_ snappkg.SnapshotSink   = (*MemorySnapshotStore)(nil)
	_ snappkg.SnapshotSource = (*MemorySnapshotStore)(nil)
	_ snappkg.SnapshotStore  = (*MemorySnapshotStore)(nil)
)

func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		Lifecycle: dispatcher.Lifecycle{},
		mu:        sync.RWMutex{},
		snapshots: make(map[string]*snappkg.Snapshot),
	}
}

func (s *MemorySnapshotStore) Save(_ context.Context, snap snappkg.Snapshot) error {
	return s.withWriteLock("memory.snapshot_save_failed", "snapshot store save", func() error {
		key := id.NewStreamRef(snap.StreamType, snap.StreamID).StreamKey()

		existing, exists := s.snapshots[key]
		if exists && existing.Version.Int() > snap.Version.Int() {
			return nil
		}

		s.snapshots[key] = copySnapshot(&snap)

		return nil
	})
}

// withWriteLock checks the store is open, acquires the write lock, and runs fn
// under the lock. Centralises the wrapClosed + Lock + defer Unlock preamble
// shared by all write-side methods.
func (s *MemorySnapshotStore) withWriteLock(code, msg string, fn func() error) error {
	if err := wrapClosed(s.CheckClosed(snappkg.ErrSnapshotStoreClosed), code, msg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn()
}

// withSnapshotReadLock is the read-side companion to withWriteLock. Top-level
// generic function because Go does not permit generic methods.
func withSnapshotReadLock[T any](
	s *MemorySnapshotStore,
	code, msg string,
	fn func() (T, error),
) (T, error) {
	if err := wrapClosed(s.CheckClosed(snappkg.ErrSnapshotStoreClosed), code, msg); err != nil {
		var zero T

		return zero, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn()
}

func (s *MemorySnapshotStore) Load(
	_ context.Context,
	ref id.StreamRef,
) (*snappkg.Snapshot, error) {
	return withSnapshotReadLock(
		s,
		"memory.snapshot_load_failed",
		"snapshot store load",
		func() (*snappkg.Snapshot, error) {
			key := ref.StreamKey()

			snap, exists := s.snapshots[key]
			if !exists {
				return nil, snappkg.ErrSnapshotNotFound
			}

			return copySnapshot(snap), nil
		},
	)
}

func (s *MemorySnapshotStore) LoadAtVersion(
	_ context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snappkg.Snapshot, error) {
	return withSnapshotReadLock(
		s,
		"memory.snapshot_load_at_version_failed",
		"snapshot store load at version",
		func() (*snappkg.Snapshot, error) {
			key := ref.StreamKey()

			snap, exists := s.snapshots[key]
			if !exists {
				return nil, snappkg.ErrSnapshotNotFound
			}

			if snap.Version.Cmp(version) > 0 {
				return nil, snappkg.ErrSnapshotNotFound
			}

			return copySnapshot(snap), nil
		},
	)
}

func copySnapshot(snap *snappkg.Snapshot) *snappkg.Snapshot {
	snapshotCopy := *snap

	if snap.State != nil {
		snapshotCopy.State = make([]byte, len(snap.State))
		copy(snapshotCopy.State, snap.State)
	}

	return &snapshotCopy
}

func (s *MemorySnapshotStore) Delete(
	_ context.Context,
	ref id.StreamRef,
) error {
	return s.withWriteLock("memory.snapshot_delete_failed", "snapshot store delete", func() error {
		delete(s.snapshots, ref.StreamKey())

		return nil
	})
}

func (s *MemorySnapshotStore) Close() error {
	return s.Lifecycle.Close()
}
