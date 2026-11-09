package snapshot_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// fakeStore is a minimal in-memory SnapshotStore for tests.
type fakeStore struct {
	mu     sync.RWMutex
	data   map[string]*snapshot.Snapshot
	closed bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string]*snapshot.Snapshot)}
}

func (f *fakeStore) key(ref id.StreamRef) string {
	return fmt.Sprintf("%s:%s", ref.Type, ref.ID)
}

func (f *fakeStore) Save(_ context.Context, snap snapshot.Snapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ref := id.NewStreamRef(snap.StreamType, snap.StreamID)
	f.data[f.key(ref)] = &snap

	return nil
}

func (f *fakeStore) Load(_ context.Context, ref id.StreamRef) (*snapshot.Snapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	s, ok := f.data[f.key(ref)]
	if !ok {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return s, nil
}

func (f *fakeStore) Delete(_ context.Context, ref id.StreamRef) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.data, f.key(ref))

	return nil
}

func (f *fakeStore) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	s, err := f.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	if s.Version != version {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return s, nil
}

func (f *fakeStore) Close() error {
	f.closed = true

	return nil
}
