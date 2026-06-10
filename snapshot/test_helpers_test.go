package snapshot_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
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

func (f *fakeStore) key(ref event.AggregateRef) string {
	return fmt.Sprintf("%s:%s", ref.Type, ref.ID)
}

func (f *fakeStore) Save(_ context.Context, snap snapshot.Snapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ref := event.NewAggregateRef(snap.AggregateType, snap.AggregateID)
	f.data[f.key(ref)] = &snap

	return nil
}

func (f *fakeStore) Load(_ context.Context, ref event.AggregateRef) (*snapshot.Snapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	s, ok := f.data[f.key(ref)]
	if !ok {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return s, nil
}

func (f *fakeStore) Delete(_ context.Context, ref event.AggregateRef) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.data, f.key(ref))

	return nil
}

func (f *fakeStore) LoadAtVersion(_ context.Context, ref event.AggregateRef, version event.Version) (*snapshot.Snapshot, error) {
	s, err := f.Load(context.TODO(), ref)
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
