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

func TestTurso_SyncRejectsMemoryDB(t *testing.T) {
	t.Parallel()

	_, err := OpenTursoSync(context.Background(), ":memory:", "https://example.com", "token")
	if err == nil {
		t.Fatal("expected error for :memory: with remote URL")
	}

	if !errors.Is(err, ErrTursoMemorySync) {
		t.Errorf("error = %v, want ErrTursoMemorySync", err)
	}
}

func TestTurso_ConstructorDelegatesToSQLite(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	_, err = store.Load(context.Background(), "Order", id.NewAggregateID())
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestTurso_EventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := tursoTestEvent(t, aggID, 1)

	err = store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
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
}

func TestTurso_EventStore_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := tursoTestEvent(t, aggID, 1)

	err = store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}

	evt2 := tursoTestEvent(t, aggID, 2)
	err = store.Save(context.Background(), "Order", aggID, []event.Event{evt2}, event.Version(0))
	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
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
}

func TestTurso_CheckpointStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewTursoCheckpointStore: %v", err)
	}

	eventID := id.NewEventID()

	err = store.Save(context.Background(), "order_projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "order_projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != eventID {
		t.Errorf("EventID = %v, want %v", loaded, eventID)
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

	err = txStore.SaveWithOutbox(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
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

func TestTurso_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, err := NewTursoEventStore(db)
	if err != nil {
		t.Fatalf("NewTursoEventStore: %v", err)
	}

	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt := tursoTestEvent(t, aggID, 1,
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", "production"),
	)

	err = store.Save(context.Background(), "Order", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Order", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
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

func TestTurso_FullWorkflow(t *testing.T) {
	t.Parallel()

	db := newTursoTestDB(t)
	initTursoSchema(t, db)

	store, _ := NewTursoEventStore(db)
	snapStore, _ := NewTursoSnapshotStore(db)
	checkpointStore, _ := NewTursoCheckpointStore(db)

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
