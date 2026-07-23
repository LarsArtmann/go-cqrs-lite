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
	if err := wrapClosed(
		s.CheckClosed(snappkg.ErrSnapshotStoreClosed),
		"memory.snapshot_save_failed",
		"snapshot store save",
	); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := id.NewAggregateRef(snap.StreamType, snap.StreamID).StreamKey()

	existing, exists := s.snapshots[key]
	if exists && existing.Version.Int() > snap.Version.Int() {
		return nil
	}

	s.snapshots[key] = copySnapshot(&snap)

	return nil
}

func (s *MemorySnapshotStore) Load(
	_ context.Context,
	ref id.StreamRef,
) (*snappkg.Snapshot, error) {
	if err := wrapClosed(
		s.CheckClosed(snappkg.ErrSnapshotStoreClosed),
		"memory.snapshot_load_failed",
		"snapshot store load",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	snap, exists := s.snapshots[key]
	if !exists {
		return nil, snappkg.ErrSnapshotNotFound
	}

	cp := copySnapshot(snap)

	return cp, nil
}

func (s *MemorySnapshotStore) LoadAtVersion(
	_ context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snappkg.Snapshot, error) {
	if err := wrapClosed(
		s.CheckClosed(snappkg.ErrSnapshotStoreClosed),
		"memory.snapshot_load_at_version_failed",
		"snapshot store load at version",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	snap, exists := s.snapshots[key]
	if !exists {
		return nil, snappkg.ErrSnapshotNotFound
	}

	if snap.Version.Cmp(version) > 0 {
		return nil, snappkg.ErrSnapshotNotFound
	}

	cp := copySnapshot(snap)

	return cp, nil
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
	if err := wrapClosed(
		s.CheckClosed(snappkg.ErrSnapshotStoreClosed),
		"memory.snapshot_delete_failed",
		"snapshot store delete",
	); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	delete(s.snapshots, key)

	return nil
}

func (s *MemorySnapshotStore) Close() error {
	return s.Lifecycle.Close()
}
