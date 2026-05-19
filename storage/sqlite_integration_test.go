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
	_ "modernc.org/sqlite"
)

func newSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	db.SetMaxOpenConns(1)

	return db
}

func initSQLiteSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteOutboxSchema()} {
		_, err := db.ExecContext(context.Background(), ddl)
		if err != nil {
			t.Fatalf("exec DDL: %v\nDDL: %s", err, ddl)
		}
	}
}

func newSQLiteTestStore(t *testing.T) *SQLEventStore {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	return store
}

func sqliteTestEvent(
	t *testing.T,
	aggID id.AggregateID,
	version event.Version,
	opts ...event.Option,
) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(
		"IssueCreated",
		aggID,
		"Issue",
		version,
		[]byte(fmt.Sprintf(`{"title":"test-%d"}`, version)),
		opts...,
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func TestSQLiteEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	evt := sqliteTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "IssueCreated" {
		t.Errorf("Type = %q, want IssueCreated", loaded[0].Type())
	}

	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID = %v, want %v", loaded[0].ID(), evt.ID())
	}

	if !loaded[0].OccurredAt().Equal(evt.OccurredAt()) {
		t.Errorf("OccurredAt = %v, want %v", loaded[0].OccurredAt(), evt.OccurredAt())
	}
}

func TestSQLiteEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	evt := sqliteTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}

	evt2 := sqliteTestEvent(t, aggID, 2)

	err = store.Save(context.Background(), "Issue", aggID, []event.Event{evt2}, event.Version(0))
	if err == nil {
		t.Fatal("expected concurrency conflict error")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf("error should wrap event.ErrVersionConflict, got: %v", err)
	}
}

func TestSQLiteEventStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := sqliteTestEvent(t, aggID, 1)
	evt2 := sqliteTestEvent(t, aggID, 2)

	err := store.AppendBatch(context.Background(), "Issue", aggID, []event.Event{evt1, evt2})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
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

func TestSQLiteEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := sqliteTestEvent(t, aggID, 1)
	evt2 := sqliteTestEvent(t, aggID, 2)
	evt3 := sqliteTestEvent(t, aggID, 3)

	err := store.AppendBatch(context.Background(), "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromVersion(context.Background(), "Issue", aggID, event.Version(1))
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

func TestSQLiteEventStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	_, err := store.Load(context.Background(), "Issue", aggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestSQLiteEventStore_Delete(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	evt := sqliteTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = store.Delete(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Load(context.Background(), "Issue", aggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound after delete, got %v", err)
	}
}

func TestSQLiteEventStore_LoadAll(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := sqliteTestEvent(
		t,
		aggID1,
		1,
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	evt2 := sqliteTestEvent(
		t,
		aggID2,
		1,
		event.WithOccurredAt(time.Now().Add(time.Second).Truncate(time.Microsecond)),
	)

	err := store.AppendBatch(context.Background(), "Issue", aggID1, []event.Event{evt1})
	if err != nil {
		t.Fatalf("AppendBatch 1: %v", err)
	}

	err = store.AppendBatch(context.Background(), "Issue", aggID2, []event.Event{evt2})
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

func TestSQLiteEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt := sqliteTestEvent(
		t, aggID, 1,
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", "test"),
	)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
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

	if meta.Custom["env"] != "test" {
		t.Errorf("Custom[env] = %q, want %q", meta.Custom["env"], "test")
	}
}

func TestSQLiteSnapshotStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       event.Version(5),
		State:         []byte(`{"title":"snapshot-issue"}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("Version = %d, want 5", loaded.Version.Int())
	}

	if string(loaded.State) != `{"title":"snapshot-issue"}` {
		t.Errorf("State = %s, want snapshot-issue", loaded.State)
	}
}

func TestSQLiteSnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       event.Version(10),
		State:         []byte(`{"title":"v10"}`),
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = store.LoadAtVersion(context.Background(), "Issue", aggID, event.Version(5))
	if !errors.Is(err, event.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound for version 5 < snapshot version 10, got %v", err)
	}

	loaded, err := store.LoadAtVersion(context.Background(), "Issue", aggID, event.Version(15))
	if err != nil {
		t.Fatalf("LoadAtVersion(15): %v", err)
	}

	if loaded.Version.Int() != 10 {
		t.Errorf("Version = %d, want 10", loaded.Version.Int())
	}
}

func TestSQLiteCheckpointStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}

	loaded, err := store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("expected zero EventID for new projection, got %v", loaded)
	}

	eventID := id.NewEventID()

	err = store.Save(context.Background(), "issue_projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err = store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != eventID {
		t.Errorf("EventID = %v, want %v", loaded, eventID)
	}

	newEventID := id.NewEventID()

	err = store.Save(context.Background(), "issue_projection", newEventID)
	if err != nil {
		t.Fatalf("Save update: %v", err)
	}

	loaded, err = store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}

	if loaded != newEventID {
		t.Errorf("EventID = %v, want %v", loaded, newEventID)
	}
}

func TestSQLiteOutbox_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	outbox, err := NewSQLiteOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLiteOutbox: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := sqliteTestEvent(t, aggID, 1)

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

	if entries[0].Events[0].Type() != "IssueCreated" {
		t.Errorf("Type = %q, want IssueCreated", entries[0].Events[0].Type())
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

func TestSQLiteTransactionalStore_SaveWithOutbox(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	outbox, err := NewSQLiteOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLiteOutbox: %v", err)
	}

	txStore, err := NewSQLiteTransactionalStore(store, outbox)
	if err != nil {
		t.Fatalf("NewSQLiteTransactionalStore: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := sqliteTestEvent(t, aggID, 1)

	err = txStore.SaveWithOutbox(
		context.Background(),
		"Issue", aggID,
		[]event.Event{evt},
		event.Version(0),
		outbox,
	)
	if err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
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

func TestNewSQLiteEventStore_NilDB(t *testing.T) {
	_, err := NewSQLiteEventStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteSnapshotStore_NilDB(t *testing.T) {
	_, err := NewSQLiteSnapshotStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteCheckpointStore_NilDB(t *testing.T) {
	_, err := NewSQLiteCheckpointStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteOutbox_NilDB(t *testing.T) {
	_, err := NewSQLiteOutbox(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteTransactionalStore_NilStore(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	outbox, err := NewSQLiteOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLiteOutbox: %v", err)
	}

	_, err = NewSQLiteTransactionalStore(nil, outbox)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestNewSQLiteTransactionalStore_NilOutbox(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	_, err = NewSQLiteTransactionalStore(store, nil)
	if err == nil {
		t.Fatal("expected error for nil outbox")
	}
}
