package turso_test

import (
	"context"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func setupEventStore(t *testing.T) (*storage.SQLEventStore, context.Context) {
	t.Helper()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewEventStore(database)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	return store, context.Background()
}

func makeEvent(t *testing.T, aggID id.AggregateID, version int) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		"test.created",
		aggID,
		"TestAggregate",
		event.Version(version),
		[]byte(`{"action":"test"}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func saveEvent(
	t *testing.T,
	store *storage.SQLEventStore,
	ctx context.Context,
	evt event.Event,
	expectedVersion event.Version,
) {
	t.Helper()

	ref := event.NewAggregateRef(evt.AggregateType(), evt.AggregateID())
	if err := store.Save(ctx, ref, []event.Event{evt}, expectedVersion); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()
	evt := makeEvent(t, aggID, 1)
	saveEvent(t, store, ctx, evt, 0)

	ref := event.NewAggregateRef("TestAggregate", aggID)
	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != "test.created" {
		t.Errorf("Type = %q, want %q", events[0].Type(), "test.created")
	}
}

func TestEventStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()
	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)

	evt2 := makeEvent(t, aggID, 2)
	evt3 := makeEvent(t, aggID, 3)
	ref := event.NewAggregateRef("TestAggregate", aggID)
	if err := store.AppendBatch(ctx, ref, []event.Event{evt2, evt3}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	for i := 1; i <= 3; i++ {
		saveEvent(t, store, ctx, makeEvent(t, aggID, i), event.Version(i-1))
	}

	ref := event.NewAggregateRef("TestAggregate", aggID)
	events, err := store.LoadFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events from version 2, got %d", len(events))
	}
}

func TestEventStore_LoadNonExistent(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	ref := event.NewAggregateRef("TestAggregate", aggID)
	_, err := store.Load(ctx, ref)
	if err == nil {
		t.Fatal("expected error for non-existent aggregate")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("expected Rejection, got %s", errorfamily.Classify(err))
	}
}

func TestEventStore_ReadAll(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()
	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 event from ReadAll, got %d", len(all))
	}
}

func TestSnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewSnapshotStore(database)
	if err != nil {
		t.Fatalf("NewSnapshotStore: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()

	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "TestAggregate",
		Version:       5,
		State:         []byte(`{"name":"test"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	ref := event.NewAggregateRef("TestAggregate", aggID)
	loaded, err := store.LoadAtVersion(ctx, ref, 5)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if loaded.AggregateID.String() != aggID.String() {
		t.Errorf("AggregateID = %q, want %q", loaded.AggregateID, aggID)
	}
}

func TestCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewCheckpointStore(database)
	if err != nil {
		t.Fatalf("NewCheckpointStore: %v", err)
	}

	ctx := context.Background()
	evtID := id.NewEventID()

	checkpoint := event.Checkpoint{EventID: evtID, ProcessedAt: time.Now()}
	if err := store.Save(ctx, "test-projection", checkpoint); err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	loaded, err := store.Load(ctx, "test-projection")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if loaded.EventID.String() != evtID.String() {
		t.Errorf("EventID = %q, want %q", loaded.EventID, evtID)
	}
}

func TestCheckpointStore_LoadEmpty(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewCheckpointStore(database)
	if err != nil {
		t.Fatalf("NewCheckpointStore: %v", err)
	}

	ctx := context.Background()
	loaded, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Load empty checkpoint: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("expected zero checkpoint, got %v", loaded)
	}
}
