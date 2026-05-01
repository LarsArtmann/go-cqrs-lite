package testhelpers

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// FakeSnapshotStore implements event.SnapshotStore for testing.
type FakeSnapshotStore struct {
	mu       sync.RWMutex
	snapshot *event.Snapshot
	saved    []event.Snapshot
	loadErr  error
	saveErr  error
}

// NewFakeSnapshotStore creates a FakeSnapshotStore with no snapshot.
func NewFakeSnapshotStore() *FakeSnapshotStore {
	return &FakeSnapshotStore{}
}

// SetSnapshot configures the snapshot returned by Load.
func (s *FakeSnapshotStore) SetSnapshot(snap *event.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = snap
}

// SetLoadError configures an error returned by Load.
func (s *FakeSnapshotStore) SetLoadError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadErr = err
}

// SetSaveError configures an error returned by Save.
func (s *FakeSnapshotStore) SetSaveError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saveErr = err
}

// Saved returns a copy of all snapshots saved via Save.
func (s *FakeSnapshotStore) Saved() []event.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]event.Snapshot{}, s.saved...)
}

// Save records the snapshot for later verification.
func (s *FakeSnapshotStore) Save(_ context.Context, snap event.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.saveErr != nil {
		return s.saveErr
	}

	s.saved = append(s.saved, snap)

	return nil
}

// Load returns the configured snapshot or error.
func (s *FakeSnapshotStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

// LoadAtVersion returns the configured snapshot or error.
func (s *FakeSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

// Delete is a no-op for testing.
func (s *FakeSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return nil
}

// Close is a no-op for testing.
func (s *FakeSnapshotStore) Close() error { return nil }

var _ event.SnapshotStore = (*FakeSnapshotStore)(nil)
