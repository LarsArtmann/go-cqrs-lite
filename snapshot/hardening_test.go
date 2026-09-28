package snapshot

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type trackingStore struct {
	mu   sync.Mutex
	last *Snapshot
}

func (s *trackingStore) Save(_ context.Context, snap Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := snap
	s.last = &cp

	return nil
}

func (s *trackingStore) Load(_ context.Context, ref id.StreamRef) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.last, nil
}

func (s *trackingStore) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	_ event.Version,
) (*Snapshot, error) {
	return s.Load(ctx, ref)
}

func (s *trackingStore) Delete(_ context.Context, _ id.StreamRef) error { return nil }

type failingStore struct{}

func (failingStore) Save(_ context.Context, _ Snapshot) error { return errStoreDown }
func (failingStore) Load(_ context.Context, _ id.StreamRef) (*Snapshot, error) {
	return nil, errStoreDown
}

func (failingStore) LoadAtVersion(
	_ context.Context,
	_ id.StreamRef,
	_ event.Version,
) (*Snapshot, error) {
	return nil, errStoreDown
}

func (failingStore) Delete(_ context.Context, _ id.StreamRef) error { return errStoreDown }

func TestReadPressure_BoundedTracking_EvictsLeastRecentlyRead(t *testing.T) {
	t.Parallel()

	rp, err := NewReadPressure(2, WithReadTrackingLimit(2))
	if err != nil {
		t.Fatalf("NewReadPressure: %v", err)
	}

	refA := id.NewStreamRef("T", id.NewStreamID())
	refB := id.NewStreamRef("T", id.NewStreamID())
	refC := id.NewStreamRef("T", id.NewStreamID())

	rp.RecordRead(refA, 1)
	rp.RecordRead(refB, 1)

	// Refresh A so B becomes the least-recently-read entry.
	rp.RecordRead(refA, 1)

	rp.RecordRead(refC, 1)

	if got := rp.ReadCount(refB); got != 0 {
		t.Fatalf("expected B evicted (least recently read), ReadCount=%d", got)
	}

	if rp.ReadCount(refA) != 2 || rp.ReadCount(refC) != 1 {
		t.Fatalf("expected A=2 C=1, got A=%d C=%d", rp.ReadCount(refA), rp.ReadCount(refC))
	}
}

func TestReadPressure_UnboundedByDefault(t *testing.T) {
	t.Parallel()

	rp, err := NewReadPressure(1)
	if err != nil {
		t.Fatalf("NewReadPressure: %v", err)
	}

	for range 10 {
		rp.RecordRead(id.NewStreamRef("T", id.NewStreamID()), 1)
	}

	rp.mu.Lock()
	defer rp.mu.Unlock()

	if len(rp.reads) != 10 {
		t.Fatalf("expected unbounded tracking to hold 10 entries, got %d", len(rp.reads))
	}
}

func TestTypedStore_Save_RejectsInvalidSnapshots(t *testing.T) {
	t.Parallel()

	store := &trackingStore{}
	ts := NewTypedStore[map[string]any](store, codec.JSONCodec{})
	ctx := context.Background()

	ref := id.NewStreamRef("Test", id.NewStreamID())

	err := ts.Save(ctx, TypedSnapshot[map[string]any]{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    0,
		State:      map[string]any{"v": 1},
	})
	if err == nil {
		t.Fatal("expected Save with version 0 to be rejected")
	}
}

func TestTypedStore_Save_StampsAndPreservesCreatedAt(t *testing.T) {
	t.Parallel()

	store := &trackingStore{}
	ts := NewTypedStore[map[string]any](store, codec.JSONCodec{})
	ctx := context.Background()

	ref := id.NewStreamRef("Test", id.NewStreamID())

	if err := ts.Save(ctx, TypedSnapshot[map[string]any]{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    3,
		State:      map[string]any{"v": 1},
	}); err != nil {
		t.Fatalf("Save (no CreatedAt): %v", err)
	}

	snap, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if snap.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be stamped by the constructor")
	}
}

// testStoreError is a SnapshotStore Save failure surfaced to verify wrapping.
var errStoreDown = errors.New("store down")

func TestTypedStore_Save_WrapsInfrastructureErrors(t *testing.T) {
	t.Parallel()

	ts := NewTypedStore[map[string]any](failingStore{}, codec.JSONCodec{})
	ctx := context.Background()

	ref := id.NewStreamRef("Test", id.NewStreamID())

	err := ts.Save(ctx, TypedSnapshot[map[string]any]{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    1,
		State:      map[string]any{"v": 1},
	})
	if err == nil || !errors.Is(err, errStoreDown) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}
