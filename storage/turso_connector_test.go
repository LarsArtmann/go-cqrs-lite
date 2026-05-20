package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newTursoTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenTurso(":memory:")
	if err != nil {
		t.Fatalf("OpenTurso: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func initTursoSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := TursoInitSchema(context.Background(), db); err != nil {
		t.Fatalf("TursoInitSchema: %v", err)
	}
}

func newTursoTestStore(t *testing.T) *SQLEventStore {
	t.Helper()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	return store
}

func tursoTestEvent(
	t *testing.T,
	aggID id.AggregateID,
	version event.Version,
	opts ...event.Option,
) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(
		"OrderPlaced",
		aggID,
		"Order",
		version,
		[]byte(fmt.Sprintf(`{"item":"widget-%d"}`, version)),
		opts...,
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func TestTurso_OpenLocalDB(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestTurso_OpenTursoInMemory(t *testing.T) {
	t.Parallel()

	db, err := OpenTursoInMemory()
	if err != nil {
		t.Fatalf("OpenTursoInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestTurso_InitSchema(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	tables := []string{"events", "snapshots", "checkpoints", "outbox"}

	for _, table := range tables {
		var name string

		err := db.QueryRowContext(
			context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name = ?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestTurso_EventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()

	evt := tursoTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "OrderPlaced" {
		t.Errorf("Type = %q, want OrderPlaced", loaded[0].Type())
	}

	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID = %v, want %v", loaded[0].ID(), evt.ID())
	}

	if !loaded[0].OccurredAt().Equal(evt.OccurredAt()) {
		t.Errorf("OccurredAt = %v, want %v", loaded[0].OccurredAt(), evt.OccurredAt())
	}
}

func TestTurso_EventStore_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()

	evt := tursoTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}

	evt2 := tursoTestEvent(t, aggID, 2)

	err = store.Save(context.Background(), "Order", aggID, []event.Event{evt2}, event.Version(0))
	if err == nil {
		t.Fatal("expected concurrency conflict error")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf("error should wrap event.ErrVersionConflict, got: %v", err)
	}
}

func TestTurso_EventStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := tursoTestEvent(t, aggID, 1)
	evt2 := tursoTestEvent(t, aggID, 2)

	err := store.AppendBatch(context.Background(), "Order", aggID, []event.Event{evt1, evt2})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	if loaded[0].Version() != 1 {
		t.Errorf("events[0].Version = %d, want 1", loaded[0].Version())
	}

	if loaded[1].Version() != 2 {
		t.Errorf("events[1].Version = %d, want 2", loaded[1].Version())
	}
}

func TestTurso_EventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := tursoTestEvent(t, aggID, 1)
	evt2 := tursoTestEvent(t, aggID, 2)
	evt3 := tursoTestEvent(t, aggID, 3)

	err := store.AppendBatch(context.Background(), "Order", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromVersion(context.Background(), "Order", aggID, event.Version(1))
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}

	if loaded[0].Version() != 2 {
		t.Errorf("events[0].Version = %d, want 2", loaded[0].Version())
	}
}

func TestTurso_EventStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)

	_, err := store.Load(context.Background(), "Order", id.NewAggregateID())
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestTurso_EventStore_Delete(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()

	evt := tursoTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = store.Delete(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Load(context.Background(), "Order", aggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound after delete, got %v", err)
	}
}

func TestTurso_EventStore_LoadAll(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := tursoTestEvent(
		t, aggID1, 1,
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	evt2 := tursoTestEvent(
		t, aggID2, 1,
		event.WithOccurredAt(time.Now().Add(time.Second).Truncate(time.Microsecond)),
	)

	err := store.AppendBatch(context.Background(), "Order", aggID1, []event.Event{evt1})
	if err != nil {
		t.Fatalf("AppendBatch 1: %v", err)
	}

	err = store.AppendBatch(context.Background(), "Order", aggID2, []event.Event{evt2})
	if err != nil {
		t.Fatalf("AppendBatch 2: %v", err)
	}

	all, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestTurso_EventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt := tursoTestEvent(
		t, aggID, 1,
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", "production"),
	)

	err := store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	meta := loaded[0].Metadata()
	if meta == nil {
		t.Fatal("Metadata is nil")
	}

	if meta.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", meta.CorrelationID, cid)
	}

	if meta.UserID != uid {
		t.Errorf("UserID = %v, want %v", meta.UserID, uid)
	}

	if meta.Custom["env"] != "production" {
		t.Errorf("Custom[env] = %q, want %q", meta.Custom["env"], "production")
	}
}

func TestTurso_EventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := tursoTestEvent(t, aggID, 1)
	evt2 := tursoTestEvent(t, aggID, 2)
	evt3 := tursoTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, "Order", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(ctx, "Order", aggID, event.Version(2))
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTurso_EventStore_LoadToVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)

	_, err := store.LoadToVersion(context.Background(), "Order", id.NewAggregateID(), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestTurso_EventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1 := tursoTestEvent(t, aggID, 1, event.WithOccurredAt(now.Add(-2*time.Hour)))
	evt2 := tursoTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(-1*time.Hour)))
	evt3 := tursoTestEvent(t, aggID, 3, event.WithOccurredAt(now))

	err := store.AppendBatch(ctx, "Order", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(ctx, "Order", aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTurso_EventStore_LoadToTimestamp_NotFound(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)

	_, err := store.LoadToTimestamp(
		context.Background(), "Order",
		id.NewAggregateID(), time.Now(),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestTurso_EventStore_LoadAllFromPosition(t *testing.T) {
	t.Parallel()

	store := newTursoTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()

	evt1 := tursoTestEvent(t, aggID, 1)

	time.Sleep(2 * time.Millisecond)
	evt2 := tursoTestEvent(t, aggID, 2)

	time.Sleep(2 * time.Millisecond)
	evt3 := tursoTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, "Order", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadAllFromPosition(ctx, evt1.ID(), 1)
	if err != nil {
		t.Fatalf("LoadAllFromPosition: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event after position, got %d", len(events))
	}

	if events[0].ID() != evt2.ID() {
		t.Fatalf("expected evt2, got event with version %d", events[0].Version())
	}

	all, err := store.LoadAllFromPosition(ctx, evt1.ID(), 0)
	if err != nil {
		t.Fatalf("LoadAllFromPosition no limit: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events after position with no limit, got %d", len(all))
	}
}

func TestTurso_SnapshotStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewTursoSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       event.Version(5),
		State:         []byte(`{"item":"turbo-widget"}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("Version = %d, want 5", loaded.Version.Int())
	}

	if string(loaded.State) != `{"item":"turbo-widget"}` {
		t.Errorf("State = %s, want turbo-widget", loaded.State)
	}
}

func TestTurso_SnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewTursoSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       event.Version(10),
		State:         []byte(`{"item":"v10"}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = store.LoadAtVersion(context.Background(), "Order", aggID, event.Version(5))
	if !errors.Is(err, event.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound for version 5 < snapshot version 10, got %v", err)
	}

	loaded, err := store.LoadAtVersion(context.Background(), "Order", aggID, event.Version(15))
	if err != nil {
		t.Fatalf("LoadAtVersion(15): %v", err)
	}

	if loaded.Version.Int() != 10 {
		t.Errorf("Version = %d, want 10", loaded.Version.Int())
	}
}

func TestTurso_SnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewTursoSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       event.Version(3),
		State:         []byte(`{}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = store.Delete(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Load(context.Background(), "Order", aggID)
	if !errors.Is(err, event.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound after delete, got %v", err)
	}
}

func TestTurso_CheckpointStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewTursoCheckpointStore: %v", err)
	}

	loaded, err := store.Load(context.Background(), "order_projection")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("expected zero EventID for new projection, got %v", loaded)
	}

	eventID := id.NewEventID()

	err = store.Save(context.Background(), "order_projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err = store.Load(context.Background(), "order_projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != eventID {
		t.Errorf("EventID = %v, want %v", loaded, eventID)
	}

	newEventID := id.NewEventID()

	err = store.Save(context.Background(), "order_projection", newEventID)
	if err != nil {
		t.Fatalf("Save update: %v", err)
	}

	loaded, err = store.Load(context.Background(), "order_projection")
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}

	if loaded != newEventID {
		t.Errorf("EventID = %v, want %v", loaded, newEventID)
	}
}

func TestTurso_Outbox_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	outbox, err := NewTursoOutbox(db)
	if err != nil {
		t.Fatalf("NewTursoOutbox: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := tursoTestEvent(t, aggID, 1)

	err = outbox.Append(context.Background(), []event.Event{evt})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := outbox.PollPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if len(entries[0].Events) != 1 {
		t.Fatalf("expected 1 event in entry, got %d", len(entries[0].Events))
	}

	if entries[0].Events[0].Type() != "OrderPlaced" {
		t.Errorf("Type = %q, want OrderPlaced", entries[0].Events[0].Type())
	}

	err = outbox.Ack(context.Background(), []event.OutboxID{entries[0].ID})
	if err != nil {
		t.Fatalf("Ack: %v", err)
	}

	entries, err = outbox.PollPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PollPending after ack: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after ack, got %d", len(entries))
	}
}

func TestTurso_TransactionalStore_SaveWithOutbox(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	outbox, err := NewTursoOutbox(db)
	if err != nil {
		t.Fatalf("NewTursoOutbox: %v", err)
	}

	txStore, err := NewTursoTransactionalStore(store, outbox)
	if err != nil {
		t.Fatalf("NewTursoTransactionalStore: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := tursoTestEvent(t, aggID, 1)

	err = txStore.SaveWithOutbox(
		context.Background(),
		"Order", aggID,
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event in store, got %d", len(loaded))
	}

	entries, err := outbox.PollPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(entries))
	}
}

func TestTurso_NilDB_Errors(t *testing.T) {
	t.Parallel()

	if _, err := NewTursoEventStore(nil); err == nil {
		t.Fatal("expected error for nil db (event store)")
	}

	if _, err := NewTursoSnapshotStore(nil); err == nil {
		t.Fatal("expected error for nil db (snapshot store)")
	}

	if _, err := NewTursoCheckpointStore(nil); err == nil {
		t.Fatal("expected error for nil db (checkpoint store)")
	}

	if _, err := NewTursoOutbox(nil); err == nil {
		t.Fatal("expected error for nil db (outbox)")
	}
}

func TestTurso_SyncRejectsMemoryDB(t *testing.T) {
	_, err := OpenTursoSync(context.Background(), ":memory:", "https://example.com", "token")
	if err == nil {
		t.Fatal("expected error for :memory: with remote URL")
	}
}

func TestTurso_FullWorkflow(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	snapStore, err := NewTursoSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewTursoSnapshotStore: %v", err)
	}

	checkpointStore, err := NewTursoCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewTursoCheckpointStore: %v", err)
	}

	aggID := id.NewAggregateID()
	ctx := context.Background()

	for i := range 5 {
		evt := tursoTestEvent(t, aggID, event.Version(i+1))
		err := store.Save(ctx, "Order", aggID, []event.Event{evt}, event.Version(i))
		if err != nil {
			t.Fatalf("Save event %d: %v", i+1, err)
		}

		if i < 4 {
			time.Sleep(time.Millisecond)
		}
	}

	events, err := store.Load(ctx, "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       event.Version(5),
		State:         []byte(`{"item":"complete-order"}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = snapStore.Save(ctx, snap)
	if err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	loadedSnap, err := snapStore.Load(ctx, "Order", aggID)
	if err != nil {
		t.Fatalf("Load snapshot: %v", err)
	}

	if loadedSnap.Version.Int() != 5 {
		t.Errorf("snapshot Version = %d, want 5", loadedSnap.Version.Int())
	}

	lastEventID := events[4].ID()
	err = checkpointStore.Save(ctx, "order_projection", lastEventID)
	if err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	checkpoint, err := checkpointStore.Load(ctx, "order_projection")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if checkpoint != lastEventID {
		t.Errorf("checkpoint = %v, want %v", checkpoint, lastEventID)
	}

	toV3, err := store.LoadToVersion(ctx, "Order", aggID, event.Version(3))
	if err != nil {
		t.Fatalf("LoadToVersion(3): %v", err)
	}

	if len(toV3) != 3 {
		t.Fatalf("expected 3 events to version 3, got %d", len(toV3))
	}

	fromPosition, err := store.LoadAllFromPosition(ctx, events[2].ID(), 10)
	if err != nil {
		t.Fatalf("LoadAllFromPosition: %v", err)
	}

	if len(fromPosition) != 2 {
		t.Fatalf("expected 2 events after position, got %d", len(fromPosition))
	}
}
