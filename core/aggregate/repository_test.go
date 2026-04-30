package aggregate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func newTestRoot() *testRoot {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	return &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}
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
		return errors.New("store unavailable") //nolint:err113 // test error
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
	bus.PublishErr = errors.New("bus unavailable") //nolint:err113 // test error
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
		return errors.New("outbox full") //nolint:err113 // test error
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

	root := &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

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

	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus(), aggregate.WithSnapshotStore(snapStore))

	root := &snapshotAwareRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

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

	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus(), aggregate.WithSnapshotStore(snapStore))
	root := &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

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
	root := &failingSnapshotRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

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
	return errors.New("deserialization failed") //nolint:err113 // test error
}

func (r *failingSnapshotRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

// failingLoadFromVersionStore wraps a FakeStore and overrides LoadFromVersion.
type failingLoadFromVersionStore struct { //nolint:embeddedstructfieldcheck // test double
	*testhelpers.FakeStore
}

func (s *failingLoadFromVersionStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("db connection lost") //nolint:err113 // test error
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

	repo := aggregate.NewRepository(store, testhelpers.NewFakeBus(), aggregate.WithSnapshotStore(snapStore))
	root := &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from LoadFromVersion failure")
	}
}
