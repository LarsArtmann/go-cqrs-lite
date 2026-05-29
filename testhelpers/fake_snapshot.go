package testhelpers

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/snapshot"
)

// FakeSnapshotStore implements snapshot.SnapshotStore for testing.
type FakeSnapshotStore struct {
	mu       sync.RWMutex
	snapshot *snapshot.Snapshot
	saved    []snapshot.Snapshot
	loadErr  error
	saveErr  error
}

// NewFakeSnapshotStore creates a FakeSnapshotStore with no snapshot.
func NewFakeSnapshotStore() *FakeSnapshotStore {
	return &FakeSnapshotStore{}
}

// SetSnapshot configures the snapshot returned by Load.
func (s *FakeSnapshotStore) SetSnapshot(snap *snapshot.Snapshot) {
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
func (s *FakeSnapshotStore) Saved() []snapshot.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]snapshot.Snapshot{}, s.saved...)
}

// Save records the snapshot for later verification.
func (s *FakeSnapshotStore) Save(_ context.Context, snap snapshot.Snapshot) error {
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
	_ event.AggregateRef,
) (*snapshot.Snapshot, error) {
	return s.loadSnapshot()
}

// LoadAtVersion returns the configured snapshot or error.
func (s *FakeSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateRef,
	_ event.Version,
) (*snapshot.Snapshot, error) {
	return s.loadSnapshot()
}

func (s *FakeSnapshotStore) loadSnapshot() (*snapshot.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

// Delete is a no-op for testing.
func (s *FakeSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateRef,
) error {
	return nil
}

// Close is a no-op for testing.
func (s *FakeSnapshotStore) Close() error { return nil }

var (
	_ snapshot.SnapshotSink   = (*FakeSnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*FakeSnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*FakeSnapshotStore)(nil)
)
