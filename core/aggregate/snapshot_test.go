package aggregate_test

import (
	"context"
	"encoding/json"
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
	core := aggregate.NewCore(orderID, orderAggregateType)

	if core.Version() != 0 {
		t.Fatalf("expected initial version 0, got %d", core.Version())
	}

	core.SetVersion(event.Version(5))

	if core.Version() != 5 {
		t.Errorf("expected version 5 after SetVersion, got %d", core.Version())
	}
}

func TestNewRepositoryWithSnapshot(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()

	repo := aggregate.NewRepositoryWithSnapshot(store, bus, snapshotStore)
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
	repo := aggregate.NewRepositoryWithSnapshot(store, bus, snapshotStore)

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
	snapshotPayload, _ := json.Marshal(map[string]string{"status": "placed"})
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

	// Snapshot state is opaque bytes; the application must deserialize it.
	// The repository only sets the version from the snapshot and replays
	// events from snapshot.Version onward. No events exist after v1 here.
	if loaded.Status() != "" {
		t.Errorf("expected empty status (snapshot state not auto-applied), got %q", loaded.Status())
	}
}

func TestEventSourcedRepository_Load_WithSnapshotAndReplay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	snapshotStore := memory.NewMemorySnapshotStore()
	repo := aggregate.NewRepositoryWithSnapshot(store, bus, snapshotStore)

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
	snapshotPayload, _ := json.Marshal(map[string]string{"status": "placed"})
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
	o2 := newOrder(orderID)
	err = repo.Load(ctx, o2)
	if err != nil {
		t.Fatalf("load order for shipping: %v", err)
	}

	err = o2.Ship(ctx)
	if err != nil {
		t.Fatalf("ship order: %v", err)
	}

	err = repo.Save(ctx, o2)
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
	repo := aggregate.NewRepositoryWithSnapshot(store, bus, snapshotStore)

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
	repo := aggregate.NewRepositoryWithSnapshot(store, bus, snapshotStore)

	orderID := id.NewAggregateID()

	// Manually save 3 events (versions 1, 2, 3)
	for i := 1; i <= 3; i++ {
		payload, _ := json.Marshal(map[string]string{"status": "placed"})
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
	snapshotPayload, _ := json.Marshal(map[string]string{"status": "placed"})
	snapshot := event.Snapshot{
		AggregateID:   orderID,
		AggregateType: orderAggregateType,
		Version:       event.Version(2),
		State:         snapshotPayload,
		CreatedAt:     time.Now(),
	}

	err := snapshotStore.Save(ctx, snapshot)
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
