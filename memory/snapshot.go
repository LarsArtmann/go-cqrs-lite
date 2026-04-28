package memory

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type MemorySnapshotStore struct {
	dispatcher.LifecycleMixin

	mu        sync.RWMutex
	snapshots map[string]*event.Snapshot
}

var _ event.SnapshotStore = (*MemorySnapshotStore)(nil)

func NewMemorySnapshotStore() *MemorySnapshotStore {
	//nolint:exhaustruct // embedded LifecycleMixin has unexported fields from different package
	return &MemorySnapshotStore{
		snapshots: make(map[string]*event.Snapshot),
	}
}

func (s *MemorySnapshotStore) streamKey(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

func (s *MemorySnapshotStore) Save(_ context.Context, snapshot event.Snapshot) error {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return errors.Wrap(err, "snapshot store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.streamKey(snapshot.AggregateType, snapshot.AggregateID)

	existing, exists := s.snapshots[key]
	if exists && existing.Version.Int() > snapshot.Version.Int() {
		return nil
	}

	s.snapshots[key] = &snapshot

	return nil
}

func (s *MemorySnapshotStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return nil, errors.Wrap(err, "snapshot store load")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.streamKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, event.ErrSnapshotNotFound
	}

	cp := *snapshot

	return &cp, nil
}

func (s *MemorySnapshotStore) LoadAtVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return nil, errors.Wrap(err, "snapshot store load at version")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.streamKey(aggregateType, aggregateID)

	snapshot, exists := s.snapshots[key]
	if !exists {
		return nil, event.ErrSnapshotNotFound
	}

	if snapshot.Version.Int() > version.Int() {
		return nil, event.ErrSnapshotNotFound
	}

	cp := *snapshot

	return &cp, nil
}

func (s *MemorySnapshotStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	err := s.CheckClosed(event.ErrSnapshotStoreClosed)
	if err != nil {
		return errors.Wrap(err, "snapshot store delete")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.streamKey(aggregateType, aggregateID)
	delete(s.snapshots, key)

	return nil
}

func (s *MemorySnapshotStore) Close() error {
	return s.LifecycleMixin.Close() //nolint:wrapcheck
}
