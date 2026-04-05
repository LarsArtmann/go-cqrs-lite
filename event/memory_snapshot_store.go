package event

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// MemorySnapshotStore is an in-memory snapshot store for testing and development.
type MemorySnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[string]*Snapshot
}

var _ SnapshotStore = (*MemorySnapshotStore)(nil)

// NewMemorySnapshotStore creates a new in-memory snapshot store.
func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		snapshots: make(map[string]*Snapshot),
	}
}

func snapshotKey(aggregateType AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

// Save persists a snapshot.
func (s *MemorySnapshotStore) Save(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := snapshotKey(snapshot.AggregateType, snapshot.AggregateID)

	existing, exists := s.snapshots[key]
	if exists && existing.Version.Int() > snapshot.Version.Int() {
		return nil
	}

	s.snapshots[key] = &snapshot

	return nil
}

// Load retrieves the latest snapshot for an aggregate.
func (s *MemorySnapshotStore) Load(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := snapshotKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, ErrSnapshotNotFound
	}

	return snapshot, nil
}

// LoadAtVersion retrieves a snapshot at or before the given version.
func (s *MemorySnapshotStore) LoadAtVersion(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
	version Version,
) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := snapshotKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, ErrSnapshotNotFound
	}

	if snapshot.Version.Int() > version.Int() {
		return nil, ErrSnapshotNotFound
	}

	return snapshot, nil
}

// Delete removes the snapshot for an aggregate.
func (s *MemorySnapshotStore) Delete(
	_ context.Context,
	aggregateType AggregateType,
	aggregateID id.AggregateID,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := snapshotKey(aggregateType, aggregateID)
	delete(s.snapshots, key)

	return nil
}
