package memory

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// MemorySnapshotStore is an in-memory implementation of event.SnapshotStore.
// It stores at most one snapshot per aggregate (the latest version).
type MemorySnapshotStore struct {
	dispatcher.Lifecycle

	mu        sync.RWMutex
	snapshots map[string]*event.Snapshot
}

var _ event.SnapshotStore = (*MemorySnapshotStore)(nil)

// NewMemorySnapshotStore creates a new in-memory snapshot store.
func NewMemorySnapshotStore() *MemorySnapshotStore {
	//nolint:exhaustruct // embedded Lifecycle has unexported fields from different package
	return &MemorySnapshotStore{
		snapshots: make(map[string]*event.Snapshot),
	}
}

// Save stores a snapshot. If a newer snapshot already exists for the aggregate, the save is silently skipped.
func (s *MemorySnapshotStore) Save(_ context.Context, snapshot event.Snapshot) error {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.snapshot_save_failed", "snapshot store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := event.StreamKey(snapshot.AggregateType, snapshot.AggregateID)

	existing, exists := s.snapshots[key]
	if exists && existing.Version.Int() > snapshot.Version.Int() {
		return nil
	}

	s.snapshots[key] = copySnapshot(&snapshot)

	return nil
}

// Load returns the latest snapshot for an aggregate.
// Returns ErrSnapshotNotFound if no snapshot exists.
func (s *MemorySnapshotStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"memory.snapshot_load_failed",
			"snapshot store load",
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, event.ErrSnapshotNotFound
	}

	cp := copySnapshot(snapshot)

	return cp, nil
}

// LoadAtVersion returns the snapshot for an aggregate if its version is at or before the requested version.
// Returns ErrSnapshotNotFound if no suitable snapshot exists.
func (s *MemorySnapshotStore) LoadAtVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"memory.snapshot_load_at_version_failed",
			"snapshot store load at version",
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, event.ErrSnapshotNotFound
	}

	if snapshot.Version.Cmp(version) > 0 {
		return nil, event.ErrSnapshotNotFound
	}

	cp := copySnapshot(snapshot)

	return cp, nil
}

func copySnapshot(snapshot *event.Snapshot) *event.Snapshot {
	snapshotCopy := *snapshot

	if snapshot.State != nil {
		snapshotCopy.State = make([]byte, len(snapshot.State))
		copy(snapshotCopy.State, snapshot.State)
	}

	return &snapshotCopy
}

// Delete removes the snapshot for an aggregate.
func (s *MemorySnapshotStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"memory.snapshot_delete_failed",
			"snapshot store delete",
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := event.StreamKey(aggregateType, aggregateID)
	delete(s.snapshots, key)

	return nil
}

// Close marks the store as closed. Subsequent operations return ErrSnapshotStoreClosed.
func (s *MemorySnapshotStore) Close() error {
	return s.Lifecycle.Close() //nolint:wrapcheck
}
