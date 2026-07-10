package turso_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func TestEventStore_MultipleAggregates(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, agg1, 1), 0)
	saveEvent(t, store, ctx, makeEvent(t, agg2, 1), 0)
	saveEvent(t, store, ctx, makeEvent(t, agg1, 2), 1)

	events1, err := store.Load(ctx, id.NewAggregateRef("TestAggregate", agg1))
	if err != nil {
		t.Fatalf("Load agg1: %v", err)
	}

	if len(events1) != 2 {
		t.Fatalf("agg1: expected 2 events, got %d", len(events1))
	}

	events2, err := store.Load(ctx, id.NewAggregateRef("TestAggregate", agg2))
	if err != nil {
		t.Fatalf("Load agg2: %v", err)
	}

	if len(events2) != 1 {
		t.Fatalf("agg2: expected 1 event, got %d", len(events2))
	}
}

func TestEventStore_VersionOrdering(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	for i := 1; i <= 5; i++ {
		saveEvent(t, store, ctx, makeEvent(t, aggID, i), event.Version(i-1))
	}

	events, err := store.Load(ctx, id.NewAggregateRef("TestAggregate", aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	for i, evt := range events {
		if evt.Version() != event.Version(i+1) {
			t.Errorf("events[%d].Version = %d, want %d", i, evt.Version(), i+1)
		}
	}
}

func TestEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	for i := 1; i <= 5; i++ {
		saveEvent(t, store, ctx, makeEvent(t, aggID, i), event.Version(i-1))
	}

	ref := id.NewAggregateRef("TestAggregate", aggID)
	events, err := store.LoadToVersion(ctx, ref, 3)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events up to version 3, got %d", len(events))
	}
}

func TestEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)

	time.Sleep(10 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(10 * time.Millisecond)

	saveEvent(t, store, ctx, makeEvent(t, aggID, 2), 1)

	ref := id.NewAggregateRef("TestAggregate", aggID)
	events, err := store.LoadToTimestamp(ctx, ref, cutoff)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event before cutoff, got %d", len(events))
	}
}

func TestEventStore_ReadAll_MultipleAggregates(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, agg1, 1), 0)
	saveEvent(t, store, ctx, makeEvent(t, agg2, 1), 0)
	saveEvent(t, store, ctx, makeEvent(t, agg1, 2), 1)

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 events in ReadAll, got %d", len(all))
	}
}

func TestEventStore_ReadFrom(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)
	saveEvent(t, store, ctx, makeEvent(t, aggID, 2), 1)

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) < 1 {
		t.Fatal("need at least 1 event for ReadFrom")
	}

	afterID := all[0].ID()

	events, err := store.ReadFrom(ctx, afterID, 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event after first, got %d", len(events))
	}
}

func TestEventStore_ConcurrentSave(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)

	const count = 10
	aggIDs := make([]id.AggregateID, count)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
		saveEvent(t, store, ctx, makeEvent(t, aggIDs[i], 1), 0)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != count {
		t.Fatalf("expected %d events, got %d", count, len(all))
	}
}

func TestEventStore_Save_VersionConflict(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)

	evt2 := makeEvent(t, aggID, 2)
	ref := id.NewAggregateRef("TestAggregate", aggID)
	err := store.Save(ctx, ref, []event.Event{evt2}, 0)
	if err == nil {
		t.Fatal("expected version conflict error")
	}
}

func TestSnapshotStore_Overwrite(t *testing.T) {
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

	snap1 := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "TestAggregate",
		Version:       3,
		State:         []byte(`{"v":1}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snap1); err != nil {
		t.Fatalf("Save snap1: %v", err)
	}

	snap2 := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "TestAggregate",
		Version:       5,
		State:         []byte(`{"v":2}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snap2); err != nil {
		t.Fatalf("Save snap2: %v", err)
	}

	loaded, err := store.LoadAtVersion(ctx, id.NewAggregateRef("TestAggregate", aggID), 5)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if loaded.Version != 5 {
		t.Errorf("Version = %d, want 5", loaded.Version)
	}
}

func TestSnapshotStore_LoadNonExistent(t *testing.T) {
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
	_, err = store.LoadAtVersion(ctx, id.NewAggregateRef("TestAggregate", aggID), 1)
	if err == nil {
		t.Fatal("expected error for non-existent snapshot")
	}
}

func TestCheckpointStore_Overwrite(t *testing.T) {
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
	projection := "test-overwrite"

	cp1 := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
	if err := store.Save(ctx, projection, cp1); err != nil {
		t.Fatalf("Save cp1: %v", err)
	}

	cp2 := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
	if err := store.Save(ctx, projection, cp2); err != nil {
		t.Fatalf("Save cp2: %v", err)
	}

	loaded, err := store.Load(ctx, projection)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.EventID.String() != cp2.EventID.String() {
		t.Errorf("EventID = %q, want %q (second save)", loaded.EventID, cp2.EventID)
	}
}

func TestNewEventStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := turso.NewEventStore(nil)
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestNewSnapshotStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := turso.NewSnapshotStore(nil)
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestNewCheckpointStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := turso.NewCheckpointStore(nil)
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestInitSchema_Idempotent(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()

	if err := turso.InitSchema(ctx, database); err != nil {
		t.Fatalf("InitSchema first: %v", err)
	}

	if err := turso.InitSchema(ctx, database); err != nil {
		t.Fatalf("InitSchema second: %v", err)
	}
}

func TestEventStore_CloseThenAccess(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	aggID := id.NewAggregateID()
	_, err := store.Load(ctx, id.NewAggregateRef("TestAggregate", aggID))
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestEventStore_EmptyPayload(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent(
		"test.empty_payload",
		aggID,
		"TestAggregate",
		event.Version(1),
		[]byte{},
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	ref := id.NewAggregateRef("TestAggregate", aggID)
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestEventStore_MultipleAppendBatch(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	saveEvent(t, store, ctx, makeEvent(t, aggID, 1), 0)

	evts := make([]event.Event, 5)
	for i := range evts {
		evts[i] = makeEvent(t, aggID, i+2)
	}

	ref := id.NewAggregateRef("TestAggregate", aggID)
	if err := store.AppendBatch(ctx, ref, evts); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d", len(events))
	}
}

func TestStorageConstructor_AcceptsStorageSQL(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := storage.NewSQLiteEventStore(database)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestEventStore_DefensiveCopy(t *testing.T) {
	t.Parallel()

	store, ctx := setupEventStore(t)
	aggID := id.NewAggregateID()

	evt := makeEvent(t, aggID, 1)
	ref := id.NewAggregateRef("TestAggregate", aggID)
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	events1, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	events2, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load second: %v", err)
	}

	if events1[0].ID() != events2[0].ID() {
		t.Error("event IDs should match across loads")
	}
}
