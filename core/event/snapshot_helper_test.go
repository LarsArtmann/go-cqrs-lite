package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type mockSnapshotStore struct {
	saved *event.Snapshot
	err   error
}

func (m *mockSnapshotStore) Save(_ context.Context, snap event.Snapshot) error {
	m.saved = &snap

	return m.err
}

func (m *mockSnapshotStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) (*event.Snapshot, error) {
	return nil, nil //nolint:nilnil
}

func (m *mockSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) (*event.Snapshot, error) {
	return nil, nil //nolint:nilnil
}

func (m *mockSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return nil
}

func (m *mockSnapshotStore) Close() error { return nil }

func TestShouldSnapshot_AllNil(t *testing.T) {
	t.Parallel()

	if event.ShouldSnapshot(nil, nil, nil, "User", event.Version(5)) {
		t.Error("expected false when all params are nil")
	}
}

func TestShouldSnapshot_NilSnapshotStore(t *testing.T) {
	t.Parallel()

	strategy := event.MustEveryNEvents(3)

	if event.ShouldSnapshot(strategy, nil, &event.JSONCodec{}, "User", event.Version(3)) {
		t.Error("expected false when snapshot store is nil")
	}
}

func TestShouldSnapshot_NilCodec(t *testing.T) {
	t.Parallel()

	strategy := event.MustEveryNEvents(3)
	store := &mockSnapshotStore{}

	if event.ShouldSnapshot(strategy, store, nil, "User", event.Version(3)) {
		t.Error("expected false when codec is nil")
	}
}

func TestShouldSnapshot_True(t *testing.T) {
	t.Parallel()

	strategy := event.MustEveryNEvents(3)
	store := &mockSnapshotStore{}

	if !event.ShouldSnapshot(strategy, store, &event.JSONCodec{}, "User", event.Version(6)) {
		t.Error("expected true when all conditions met and version is multiple")
	}
}

func TestShouldSnapshot_VersionNotMultiple(t *testing.T) {
	t.Parallel()

	strategy := event.MustEveryNEvents(3)
	store := &mockSnapshotStore{}

	if event.ShouldSnapshot(strategy, store, &event.JSONCodec{}, "User", event.Version(4)) {
		t.Error("expected false when version is not a multiple of interval")
	}
}

func TestSaveSnapshot_Success(t *testing.T) {
	t.Parallel()

	store := &mockSnapshotStore{}
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	err := event.SaveSnapshot(
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

	if store.saved == nil {
		t.Fatal("expected snapshot to be saved")
	}

	if store.saved.Version.Int() != 5 {
		t.Errorf("expected version 5, got %d", store.saved.Version.Int())
	}

	if store.saved.AggregateType != "User" {
		t.Errorf("expected aggregate type 'User', got %s", store.saved.AggregateType)
	}
}
