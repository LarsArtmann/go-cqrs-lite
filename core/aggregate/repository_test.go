package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// --- Minimal fake implementations (no memory module dependency) ---

type fakeStore struct {
	mu     sync.RWMutex
	events map[string][]event.Event
	saveFn func(
		ctx context.Context,
		aggregateType event.AggregateType,
		aggregateID id.AggregateID,
		events []event.Event,
		expectedVersion event.Version,
	) error
}

func newFakeStore() *fakeStore {
	return &fakeStore{events: make(map[string][]event.Event)}
}

func (s *fakeStore) Save(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	_ event.Version,
) error {
	if s.saveFn != nil {
		return s.saveFn( //nolint:contextcheck // test double
			context.TODO(),
			aggregateType,
			aggregateID,
			events,
			0,
		)
	}

	s.mu.Lock()

	key := string(aggregateType) + "/" + aggregateID.String()
	s.events[key] = append(s.events[key], events...)

	s.mu.Unlock()

	return nil
}

func (s *fakeStore) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	s.mu.Lock()

	key := string(aggregateType) + "/" + aggregateID.String()
	s.events[key] = append(s.events[key], events...)

	s.mu.Unlock()

	return nil
}

func (s *fakeStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	s.mu.RLock()

	key := string(aggregateType) + "/" + aggregateID.String()
	evts := s.events[key]

	s.mu.RUnlock()

	return evts, nil
}

func (s *fakeStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	s.mu.RLock()

	key := string(aggregateType) + "/" + aggregateID.String()
	all := s.events[key]

	s.mu.RUnlock()

	for i, evt := range all {
		if evt.Version() > version.Int() {
			return all[i:], nil
		}
	}

	return nil, nil
}

func (s *fakeStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	s.mu.Lock()

	key := string(aggregateType) + "/" + aggregateID.String()
	delete(s.events, key)

	s.mu.Unlock()

	return nil
}

type fakeBus struct {
	mu         sync.Mutex
	published  []event.Event
	publishErr error
}

func (b *fakeBus) Publish(_ context.Context, events ...event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.publishErr != nil {
		return b.publishErr
	}

	b.published = append(b.published, events...)

	return nil
}

func (b *fakeBus) Subscribe(_ event.Type, _ event.Handler) error { return nil }

func (b *fakeBus) SubscribeAll(_ event.Handler) error { return nil }

type fakeSnapshotStore struct {
	mu       sync.RWMutex
	snapshot *event.Snapshot
	loadErr  error
}

func (s *fakeSnapshotStore) Save(_ context.Context, _ event.Snapshot) error {
	return nil
}

func (s *fakeSnapshotStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

func (s *fakeSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

func (s *fakeSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return nil
}

type fakeOutbox struct {
	mu       sync.Mutex
	entries  []event.OutboxEntry
	appendFn func(events []event.Event) error
}

func (o *fakeOutbox) Append(_ context.Context, events []event.Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.appendFn != nil {
		return o.appendFn(events)
	}

	o.entries = append(o.entries, event.OutboxEntry{
		ID:     event.OutboxID(fmt.Sprintf("outbox-%d", len(o.entries))),
		Events: events,
	})

	return nil
}

func (o *fakeOutbox) PollPending(_ context.Context, _ int) ([]event.OutboxEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.entries, nil
}

func (o *fakeOutbox) Ack(_ context.Context, _ []event.OutboxID) error {
	o.mu.Lock()
	o.entries = nil
	o.mu.Unlock()

	return nil
}

// --- Repository unit tests ---

func newTestRoot() *testRoot {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	return &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}
}

func makeUserEvent(t *testing.T, _ int) *event.Core {
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

	store := newFakeStore()
	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestNewRepository_WithOptions(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{}
	snapStore := &fakeSnapshotStore{}
	outbox := &fakeOutbox{}

	repo := aggregate.NewRepository(
		store,
		bus,
		aggregate.WithSnapshotStore(snapStore),
		aggregate.WithOutbox(outbox),
	)

	if repo == nil {
		t.Fatal("expected non-nil repository with options")
	}
}

func TestRepository_Save_NoChanges(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)
	root := newTestRoot()

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Errorf("save with no changes should return nil, got %v", err)
	}
}

func TestRepository_Save_PublishesToBus(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t, 1))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bus.published) != 1 {
		t.Errorf("expected 1 published event, got %d", len(bus.published))
	}

	changes := root.UncommittedChanges()

	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes after save, got %d", len(changes))
	}
}

func TestRepository_Save_StoreError(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.saveFn = func(
		_ context.Context,
		_ event.AggregateType,
		_ id.AggregateID,
		_ []event.Event,
		_ event.Version,
	) error {
		return errors.New("store unavailable")
	}

	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t, 1))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from store failure")
	}
}

func TestRepository_Save_BusPublishError(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{publishErr: errors.New("bus unavailable")}
	repo := aggregate.NewRepository(store, bus)
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t, 1))

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

	store := newFakeStore()
	bus := &fakeBus{}
	outbox := &fakeOutbox{}
	repo := aggregate.NewRepository(store, bus, aggregate.WithOutbox(outbox))
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t, 1))

	err := repo.Save(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bus.published) != 0 {
		t.Errorf("outbox mode should not publish to bus, got %d published", len(bus.published))
	}

	if len(outbox.entries) != 1 {
		t.Errorf("expected 1 outbox entry, got %d", len(outbox.entries))
	}
}

func TestRepository_Save_OutboxAppendError(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{}
	outbox := &fakeOutbox{
		appendFn: func(_ []event.Event) error {
			return errors.New("outbox full")
		},
	}
	repo := aggregate.NewRepository(store, bus, aggregate.WithOutbox(outbox))
	root := newTestRoot()

	root.RecordEvent(context.Background(), makeUserEvent(t, 1))

	err := repo.Save(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from outbox append failure")
	}
}

func TestRepository_Load(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)

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

	store := newFakeStore()
	bus := &fakeBus{}
	repo := aggregate.NewRepository(store, bus)
	root := newTestRoot()

	err := repo.Load(context.Background(), root)
	if err != nil {
		t.Errorf("load from empty store should succeed with 0 events, got %v", err)
	}
}

func TestRepository_Load_WithSnapshot(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	store := newFakeStore()
	bus := &fakeBus{}
	snapStore := &fakeSnapshotStore{
		snapshot: &event.Snapshot{
			AggregateID:   aggID,
			AggregateType: "User",
			Version:       3,
			State:         []byte(`{"status":"active"}`),
		},
	}

	evt, _ := event.NewEvent("UserUpdated", aggID, "User", 4, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 3)

	repo := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapStore))

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
	store := newFakeStore()
	bus := &fakeBus{}
	snapStore := &fakeSnapshotStore{snapshot: nil}

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(context.Background(), "User", aggID, []event.Event{evt}, 0)

	repo := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapStore))
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
	store := newFakeStore()
	bus := &fakeBus{}
	snapStore := &fakeSnapshotStore{
		snapshot: &event.Snapshot{
			AggregateID:   aggID,
			AggregateType: "User",
			Version:       2,
			State:         []byte(`{}`),
		},
	}

	repo := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapStore))
	root := &failingSnapshotRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from ApplySnapshot failure")
	}
}

func TestRepository_Load_LoadFromVersionError(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	store := newFailingLoadFromVersionStore()
	bus := &fakeBus{}
	snapStore := &fakeSnapshotStore{
		snapshot: &event.Snapshot{
			AggregateID:   aggID,
			AggregateType: "User",
			Version:       2,
			State:         []byte(`{}`),
		},
	}

	repo := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapStore))
	root := &testRoot{Core: aggregate.NewCore(aggID, event.AggregateType("User"))}

	err := repo.Load(context.Background(), root)
	if err == nil {
		t.Fatal("expected error from LoadFromVersion failure")
	}
}

// --- Additional test helpers ---

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

type failingLoadFromVersionStore struct {
	*fakeStore

	loadFromVersionErr error
}

func newFailingLoadFromVersionStore() *failingLoadFromVersionStore {
	return &failingLoadFromVersionStore{
		fakeStore:          newFakeStore(),
		loadFromVersionErr: errors.New("db connection lost"),
	}
}

func (s *failingLoadFromVersionStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, s.loadFromVersionErr
}
