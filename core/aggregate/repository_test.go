package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func newTestRoot() *testRoot {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	return &testRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}
}

func makeUserEvent(t *testing.T) *event.Core {
	t.Helper()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("makeUserEvent: %v", err)
	}

	return evt
}

func TestNewRepository(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(testhelpers.NewFakeStore(), testhelpers.NewFakeBus())

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestNewRepository_WithOptions(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	outbox := testhelpers.NewFakeOutbox()

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithOutbox(outbox),
	)

	if repo == nil {
		t.Fatal("expected non-nil repository with options")
	}
}

func TestRepository_Save_NoChanges(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(testhelpers.NewFakeStore(), testhelpers.NewFakeBus())

	err := repo.Save(context.Background(), newTestRoot())
	if err != nil {
		t.Errorf("save with no changes should return nil, got %v", err)
	}
}

func TestRepository_Save_PublishesToBus(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	repo := aggregate.NewRepository(testhelpers.NewFakeStore(), bus)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bus.Published) != 1 {
		t.Errorf("expected 1 published event, got %d", len(bus.Published))
	}

	changes := root.UncommittedChanges()

	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes after save, got %d", len(changes))
	}
}

func TestRepository_Save_StoreError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	store.SaveFn(func(
		_ context.Context,
		_ event.AggregateType,
		_ id.AggregateID,
		_ []event.Event,
		_ event.Version,
	) error {
		return errors.New("store unavailable")
	})

	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus())
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from store failure")
	}
}

func TestRepository_Save_BusPublishError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	bus.PublishErr = errors.New("bus unavailable")
	repo := aggregate.NewRepository(testhelpers.NewFakeStore(), bus)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from bus publish failure")
	}

	changes := root.UncommittedChanges()

	if len(changes) != 1 {
		t.Errorf("expected 1 uncommitted change after bus error, got %d", len(changes))
	}
}

func TestRepository_Save_WithOutbox(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	outbox := testhelpers.NewFakeOutbox()
	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		bus,
		aggregate.WithOutbox(outbox),
	)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bus.Published) != 0 {
		t.Errorf("outbox mode should not publish to bus, got %d published", len(bus.Published))
	}

	if len(outbox.Entries) != 1 {
		t.Errorf("expected 1 outbox entry, got %d", len(outbox.Entries))
	}
}

func TestRepository_Save_OutboxAppendError(t *testing.T) {
	t.Parallel()

	outbox := testhelpers.NewFakeOutbox()
	outbox.AppendFn(func(_ []event.Event) error {
		return errors.New("outbox full")
	})

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithOutbox(outbox),
	)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from outbox append failure")
	}
}

func TestRepository_Load(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus())

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 0)

	root := &testRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root.Version() != 1 {
		t.Errorf("expected version 1 after load, got %d", root.Version())
	}
}

func TestRepository_Load_EmptyStore(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(testhelpers.NewFakeStore(), testhelpers.NewFakeBus())

	err := repo.Load(context.Background(), newTestRoot())
	if err != nil {
		t.Errorf("load from empty store should succeed with 0 events, got %v", err)
	}
}

func TestRepository_Load_WithSnapshot(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	store := testhelpers.NewFakeStore()
	snapStore := testhelpers.NewFakeSnapshotStore()
	snapStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       3,
		State:         []byte(`{"status":"active"}`),
	})

	evt, _ := event.NewEvent("UserUpdated", aggID, "User", 4, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 3)

	repo := aggregate.NewRepository(
		store,
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
	)

	root := &snapshotAwareRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !root.snapshotApplied {
		t.Error("expected snapshot to be applied")
	}

	if root.Version() < 3 {
		t.Errorf("expected version >= 3 from snapshot, got %d", root.Version())
	}
}

func TestRepository_Load_SnapshotNotFound(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	store := testhelpers.NewFakeStore()
	snapStore := testhelpers.NewFakeSnapshotStore()

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 0)

	repo := aggregate.NewRepository(
		store,
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
	)
	root := &testRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root.Version() != 1 {
		t.Errorf("expected version 1 from full replay (no snapshot), got %d", root.Version())
	}
}

func TestRepository_Load_SnapshotApplyError(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	snapStore := testhelpers.NewFakeSnapshotStore()
	snapStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       2,
		State:         []byte(`{}`),
	})

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
	)
	root := &failingSnapshotRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from ApplySnapshot failure")
	}
}

// --- Test-specific root types ---

type snapshotAwareRoot struct {
	*aggregate.Core

	snapshotApplied bool
}

func (r *snapshotAwareRoot) Apply(_ event.Event) error { return nil }

func (r *snapshotAwareRoot) ApplySnapshot(_ []byte) error {
	r.snapshotApplied = true

	return nil
}

func (r *snapshotAwareRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

type failingSnapshotRoot struct {
	*aggregate.Core
}

func (r *failingSnapshotRoot) Apply(_ event.Event) error { return nil }

func (r *failingSnapshotRoot) ApplySnapshot(_ []byte) error {
	return errors.New("deserialization failed")
}

func (r *failingSnapshotRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

type serializableRoot struct {
	IDVal   id.AggregateID      `json:"id"`
	TypeVal event.AggregateType `json:"type"`
	Ver     int                 `json:"version"`
	Name    string              `json:"name"`
	changes []event.Event
}

func (r *serializableRoot) ID() id.AggregateID           { return r.IDVal }
func (r *serializableRoot) Type() event.AggregateType    { return r.TypeVal }
func (r *serializableRoot) Version() event.Version       { return event.Version(r.Ver) }
func (r *serializableRoot) SetVersion(v event.Version)   { r.Ver = v.Int() }
func (r *serializableRoot) Apply(_ event.Event) error    { return nil }
func (r *serializableRoot) ApplySnapshot(_ []byte) error { return nil }
func (r *serializableRoot) UncommittedChanges() []event.Event {
	return append([]event.Event{}, r.changes...)
}
func (r *serializableRoot) MarkChangesAsCommitted() { r.changes = r.changes[:0] }
func (r *serializableRoot) LoadEvents(events []event.Event) error {
	for _, evt := range events {
		err := r.Apply(evt)
		if err != nil {
			return fmt.Errorf("apply event %s: %w", evt.Type(), err)
		}

		r.Ver++
	}

	return nil
}

func (r *serializableRoot) recordEvent(_ context.Context, evt event.Event) {
	r.changes = append(r.changes, evt)
	r.Ver++
}

// failingLoadFromVersionStore wraps a FakeStore and overrides LoadFromVersion.
type failingLoadFromVersionStore struct {
	*testhelpers.FakeStore
}

func (s *failingLoadFromVersionStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("db connection lost")
}

func TestRepository_Load_LoadFromVersionError(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	store := &failingLoadFromVersionStore{FakeStore: testhelpers.NewFakeStore()}
	snapStore := testhelpers.NewFakeSnapshotStore()
	snapStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       2,
		State:         []byte(`{}`),
	})

	repo := aggregate.NewRepository(
		store,
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
	)
	root := &testRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from LoadFromVersion failure")
	}
}

func TestEveryNEvents(t *testing.T) {
	t.Parallel()

	strategy := aggregate.EveryNEvents(5)

	tests := []struct {
		version  int
		expected bool
	}{
		{0, false},
		{1, false},
		{4, false},
		{5, true},
		{6, false},
		{9, false},
		{10, true},
		{15, true},
	}

	for _, tc := range tests {
		got := strategy.ShouldSnapshot("User", event.Version(tc.version))
		if got != tc.expected {
			t.Errorf("EveryNEvents(5).ShouldSnapshot(User, %d) = %v, want %v",
				tc.version, got, tc.expected)
		}
	}
}

func TestRepository_Save_CreatesSnapshot(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	store := testhelpers.NewFakeStore()
	repo := aggregate.NewRepository(
		store,
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(1)),
		aggregate.WithCodec(event.JSONCodec{}),
	)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	root := &serializableRoot{
		IDVal:   aggID,
		TypeVal: event.AggregateType("User"),
		Name:    "Alice",
	}
	root.recordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved := snapStore.Saved()
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved snapshot, got %d", len(saved))
	}

	if saved[0].AggregateType != "User" {
		t.Errorf("expected aggregate type User, got %s", saved[0].AggregateType)
	}

	if saved[0].Version != 1 {
		t.Errorf("expected snapshot version 1, got %d", saved[0].Version)
	}

	if len(saved[0].State) == 0 {
		t.Error("expected non-empty snapshot state")
	}
}

func TestRepository_Save_NoSnapshotWithoutStrategy(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
	)

	root := newTestRoot()
	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved := snapStore.Saved()
	if len(saved) != 0 {
		t.Errorf("expected 0 snapshots without strategy, got %d", len(saved))
	}
}

func TestRepository_Save_SnapshotWithCodec(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(1)),
		aggregate.WithCodec(event.JSONCodec{}),
	)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	root := &serializableRoot{
		IDVal:   aggID,
		TypeVal: event.AggregateType("User"),
		Name:    "Alice",
	}
	root.recordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved := snapStore.Saved()
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved snapshot, got %d", len(saved))
	}

	if len(saved[0].State) == 0 {
		t.Error("expected non-empty state from codec encoding")
	}
}

func TestRepository_Save_SnapshotStoreError(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	snapStore.SetSaveError(errors.New("disk full"))

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(1)),
		aggregate.WithCodec(event.JSONCodec{}),
	)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	root := &serializableRoot{
		IDVal:   aggID,
		TypeVal: event.AggregateType("User"),
		Name:    "Alice",
	}
	root.recordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from snapshot save failure")
	}
}

func TestRepository_Save_NoSnapshotWithoutStore(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(1)),
	)

	root := newTestRoot()
	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepository_Save_NoSnapshotWithoutCodec(t *testing.T) {
	t.Parallel()

	snapStore := testhelpers.NewFakeSnapshotStore()
	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(1)),
	)

	root := newTestRoot()
	root.RecordEvent(context.Background(), makeUserEvent(t))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved := snapStore.Saved()
	if len(saved) != 0 {
		t.Errorf("expected 0 snapshots without codec, got %d", len(saved))
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 0)

	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus())
	root := &testRoot{Core: aggregate.MustNewCore(aggID, event.AggregateType("User"))}

	err := repo.Delete(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, _ := store.Load(context.Background(), "User", aggID)
	if len(events) != 0 {
		t.Errorf("expected 0 events after delete, got %d", len(events))
	}
}

func TestRepository_Delete_StoreError(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(
		&failingDeleteStore{FakeStore: testhelpers.NewFakeStore()},
		testhelpers.NewFakeBus(),
	)
	root := newTestRoot()

	err := repo.Delete(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from store delete failure")
	}
}

type failingDeleteStore struct {
	*testhelpers.FakeStore
}

func (s *failingDeleteStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return errors.New("connection lost")
}

func TestNewRepository_WithCodec(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithCodec(event.JSONCodec{}),
	)

	if repo == nil {
		t.Fatal("expected non-nil repository with codec")
	}
}

func TestNewRepository_WithSnapshotStrategy(t *testing.T) {
	t.Parallel()

	repo := aggregate.NewRepository(
		testhelpers.NewFakeStore(),
		testhelpers.NewFakeBus(),
		aggregate.WithSnapshotStrategy(aggregate.EveryNEvents(10)),
	)

	if repo == nil {
		t.Fatal("expected non-nil repository with snapshot strategy")
	}
}
