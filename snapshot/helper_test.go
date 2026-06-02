package snapshot_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

type mockSink struct {
	saved []snapshot.Snapshot
}

func (m *mockSink) Save(_ context.Context, snap snapshot.Snapshot) error {
	m.saved = append(m.saved, snap)

	return nil
}

func (m *mockSink) Delete(_ context.Context, _ event.AggregateRef) error { return nil }

func (m *mockSink) Close() error { return nil }

func TestShouldSnapshot_AllNil(t *testing.T) {
	t.Parallel()

	if snapshot.ShouldSnapshot(nil, nil, nil, "User", event.Version(5)) {
		t.Error("expected false when all params are nil")
	}
}

func TestShouldSnapshot_NilSnapshotStore(t *testing.T) {
	t.Parallel()

	strategy := snapshot.MustEveryNEvents(3)

	if snapshot.ShouldSnapshot(strategy, nil, &codec.JSONCodec{}, "User", event.Version(3)) {
		t.Error("expected false when snapshot store is nil")
	}
}

func TestShouldSnapshot_NilCodec(t *testing.T) {
	t.Parallel()

	strategy := snapshot.MustEveryNEvents(3)
	store := &mockSink{}

	if snapshot.ShouldSnapshot(strategy, store, nil, "User", event.Version(3)) {
		t.Error("expected false when codec is nil")
	}
}

func TestShouldSnapshot_True(t *testing.T) {
	t.Parallel()

	strategy := snapshot.MustEveryNEvents(3)
	store := &mockSink{}

	if !snapshot.ShouldSnapshot(strategy, store, &codec.JSONCodec{}, "User", event.Version(6)) {
		t.Error("expected true when all conditions met and version is multiple")
	}
}

func TestShouldSnapshot_VersionNotMultiple(t *testing.T) {
	t.Parallel()

	strategy := snapshot.MustEveryNEvents(3)
	store := &mockSink{}

	if snapshot.ShouldSnapshot(strategy, store, &codec.JSONCodec{}, "User", event.Version(4)) {
		t.Error("expected false when version is not a multiple of interval")
	}
}

func TestSaveSnapshot_Success(t *testing.T) {
	t.Parallel()

	store := &mockSink{}
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	err := snapshot.SaveSnapshot(
		context.Background(),
		store,
		"User",
		aggID,
		event.Version(5),
		[]byte(`{"name":"John"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.saved) != 1 {
		t.Fatalf("expected 1 snapshot to be saved, got %d", len(store.saved))
	}

	if store.saved[0].Version != 5 {
		t.Errorf("expected version 5, got %d", store.saved[0].Version)
	}

	if store.saved[0].AggregateType != "User" {
		t.Errorf("expected aggregate type 'User', got %s", store.saved[0].AggregateType)
	}
}
