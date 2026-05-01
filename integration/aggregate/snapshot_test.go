package aggregate_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestSetVersion(t *testing.T) {
	t.Parallel()

	orderID := id.NewAggregateID()
	core := aggregate.MustNewCore(orderID, orderAggregateType)

	if core.Version() != 0 {
		t.Fatalf("expected initial version 0, got %d", core.Version())
	}

	core.SetVersion(event.Version(5))

	if core.Version() != 5 {
		t.Errorf("expected version 5 after SetVersion, got %d", core.Version())
	}
}

func TestNewRepository_WithSnapshotStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()

	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestEventSourcedRepository_Load_WithSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	// Place order (creates OrderPlaced event at version 1)
	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	// Save the order
	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	// Save a snapshot at version 1
	snapshotPayload, err := json.Marshal(map[string]string{"status": "placed"})
	if err != nil {
		t.Fatalf("marshal snapshot payload: %v", err)
	}

	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(1),
		State:         snapshotPayload,
		CreatedAt:     time.Now(),
	}

	err = snapshotStore.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Load the order back via repository (should use snapshot)
	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load order with snapshot: %v", err)
	}

	if loaded.Version() != 1 {
		t.Errorf("expected version 1 after snapshot load, got %d", loaded.Version())
	}

	// Snapshot state is applied via Root.ApplySnapshot, then events from
	// snapshot.Version onward are replayed. No events exist after v1 here.
	if loaded.Status() != "placed" {
		t.Errorf("expected status 'placed' from snapshot, got %q", loaded.Status())
	}
}

func TestEventSourcedRepository_Load_WithSnapshotAndReplay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	// Place order (version 1)
	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	// Save snapshot at version 1
	snapshotPayload, err := json.Marshal(map[string]string{"status": "placed"})
	if err != nil {
		t.Fatalf("marshal snapshot payload: %v", err)
	}

	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(1),
		State:         snapshotPayload,
		CreatedAt:     time.Now(),
	}

	err = snapshotStore.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Ship order (version 2) — saved AFTER snapshot
	orderForShipping := newOrder(orderID)

	err = repo.Load(ctx, orderForShipping)
	if err != nil {
		t.Fatalf("load order for shipping: %v", err)
	}

	err = orderForShipping.Ship(ctx)
	if err != nil {
		t.Fatalf("ship order: %v", err)
	}

	err = repo.Save(ctx, orderForShipping)
	if err != nil {
		t.Fatalf("save shipped order: %v", err)
	}

	// Load again — should use snapshot at v1 + replay v2
	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load order with snapshot+replay: %v", err)
	}

	if loaded.Version() != 2 {
		t.Errorf("expected version 2 after snapshot+replay, got %d", loaded.Version())
	}

	if loaded.Status() != "shipped" {
		t.Errorf("expected status 'shipped' after snapshot+replay, got %q", loaded.Status())
	}
}

func TestEventSourcedRepository_Load_SnapshotNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	// Place and save order (no snapshot)
	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	// Load — snapshot store is empty, should fallback to loading all events
	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load order without snapshot: %v", err)
	}

	if loaded.Version() != 1 {
		t.Errorf("expected version 1, got %d", loaded.Version())
	}

	if loaded.Status() != "placed" {
		t.Errorf("expected status 'placed', got %q", loaded.Status())
	}
}

func TestEventSourcedRepository_Load_SnapshotLoadsFromVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))

	orderID := id.NewAggregateID()

	// Manually save 3 events (versions 1, 2, 3)
	for i := 1; i <= 3; i++ {
		payload, err := json.Marshal(map[string]string{"status": "placed"})
		if err != nil {
			t.Fatalf("marshal payload for event %d: %v", i, err)
		}

		evt, err := event.NewEvent("OrderPlaced", orderID, orderAggregateType, i, payload)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		err = store.Save(ctx, orderAggregateType, orderID, []event.Event{evt}, event.Version(i-1))
		if err != nil {
			t.Fatalf("save event %d: %v", i, err)
		}
	}

	// Save snapshot at version 2
	snapshotPayload, err := json.Marshal(map[string]string{"status": "placed"})
	if err != nil {
		t.Fatalf("marshal snapshot payload: %v", err)
	}

	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(2),
		State:         snapshotPayload,
		CreatedAt:     time.Now(),
	}

	err = snapshotStore.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Load — should start from snapshot v2 and replay only event v3
	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load with snapshot from version: %v", err)
	}

	if loaded.Version() != 3 {
		t.Errorf("expected version 3 (snapshot v2 + 1 replayed event), got %d", loaded.Version())
	}
}

type failingSnapshotRoot struct {
	*aggregate.Core
}

func (r *failingSnapshotRoot) Apply(_ event.Event) error { return nil }

func (r *failingSnapshotRoot) ApplySnapshot(_ []byte) error {
	return errors.New("apply snapshot failed")
}

func (r *failingSnapshotRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

var _ aggregate.Root = (*failingSnapshotRoot)(nil)

func TestEventSourcedRepository_Load_SnapshotApplyError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo, _ := aggregate.NewRepository(store, bus, aggregate.WithSnapshotStore(snapshotStore))

	orderID := id.NewAggregateID()

	// Save a snapshot
	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(1),
		State:         []byte(`{"status":"placed"}`),
		CreatedAt:     time.Now(),
	}

	err := snapshotStore.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Load with a root that fails ApplySnapshot
	loaded := &failingSnapshotRoot{Core: aggregate.MustNewCore(orderID, orderAggregateType)}

	err = repo.Load(ctx, loaded)
	if err == nil {
		t.Fatal("expected error when ApplySnapshot fails")
	}
}

func TestEventSourcedRepository_Load_LoadFromVersionError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()

	// Use failingStore for LoadFromVersion error
	repo, _ := aggregate.NewRepository(
		&failingStore{},
		bus,
		aggregate.WithSnapshotStore(snapshotStore),
	)

	orderID := id.NewAggregateID()

	// Save a snapshot
	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(1),
		State:         []byte(`{"status":"placed"}`),
		CreatedAt:     time.Now(),
	}

	err := snapshotStore.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Load — snapshot succeeds but LoadFromVersion fails
	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err == nil {
		t.Fatal("expected error when LoadFromVersion fails after snapshot")
	}
}
