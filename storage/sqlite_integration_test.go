package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
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

	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteSagaSchema()} {
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

func TestSQLiteEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testEventStore_SaveAndLoad(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()
	testEventStore_ConcurrencyConflict(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_AppendBatch(t *testing.T) {
	t.Parallel()
	testEventStore_AppendBatch(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()
	testEventStore_LoadFromVersion(t, newSQLiteTestStore(t), issueStoreConfig())
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

	evt := issueStoreConfig().newTestEvent(t, aggID, 1)

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

	evt1 := issueStoreConfig().newTestEvent(
		t,
		aggID1,
		1,
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	evt2 := issueStoreConfig().newTestEvent(
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
	testEventStore_MetadataRoundtrip(t, newSQLiteTestStore(t), issueStoreConfig(), "test")
}

func newTestSnapshot(aggID id.AggregateID, version event.Version, state []byte) event.Snapshot {
	return event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
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
	snap := newTestSnapshot(aggID, 5, []byte(`{"title":"snapshot-issue"}`))

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
	snap := newTestSnapshot(aggID, 10, []byte(`{"title":"v10"}`))

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

	testOutbox_Roundtrip(t, outbox, func() event.Event {
		return issueStoreConfig().newTestEvent(t, id.NewAggregateID(), 1)
	})
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

	txStore, err := NewSQLTransactionalStore(store, outbox)
	if err != nil {
		t.Fatalf("NewSQLTransactionalStore: %v", err)
	}

	aggID := id.NewAggregateID()
	evt := issueStoreConfig().newTestEvent(t, aggID, 1)

	err = txStore.SaveWithOutbox(
		context.Background(),
		"Issue", aggID,
		[]event.Event{evt},
		event.Version(0),
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

func TestNewSQLTransactionalStore_NilStore_SQLite(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	outbox, err := NewSQLiteOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLiteOutbox: %v", err)
	}

	_, err = NewSQLTransactionalStore(nil, outbox)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestNewSQLTransactionalStore_NilOutbox_SQLite(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	_, err = NewSQLTransactionalStore(store, nil)
	if err == nil {
		t.Fatal("expected error for nil outbox")
	}
}

func newSQLiteTestSagaStore(t *testing.T) *SQLSagaStore {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSagaStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSagaStore: %v", err)
	}

	return store
}

func TestSQLiteSagaStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestSagaStore(t)
	ctx := context.Background()

	state := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusRunning,
		CurrentStep: 2,
		ErrMsg:      "test error",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, state.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != state.ID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, state.ID)
	}
	if loaded.SagaType != state.SagaType {
		t.Errorf("SagaType mismatch: got %q, want %q", loaded.SagaType, state.SagaType)
	}
	if loaded.Status != state.Status {
		t.Errorf("Status mismatch: got %q, want %q", loaded.Status, state.Status)
	}
	if loaded.CurrentStep != state.CurrentStep {
		t.Errorf("CurrentStep mismatch: got %d, want %d", loaded.CurrentStep, state.CurrentStep)
	}
	if loaded.ErrMsg != state.ErrMsg {
		t.Errorf("ErrMsg mismatch: got %q, want %q", loaded.ErrMsg, state.ErrMsg)
	}
}

func TestSQLiteSagaStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestSagaStore(t)
	ctx := context.Background()

	_, err := store.Load(ctx, id.NewAggregateID())
	if !errors.Is(err, saga.ErrSagaNotFound) {
		t.Fatalf("expected ErrSagaNotFound, got: %v", err)
	}
}

func TestSQLiteSagaStore_LoadAllRunning(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestSagaStore(t)
	ctx := context.Background()

	// Create running saga
	running := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusRunning,
		CurrentStep: 1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Save(ctx, running); err != nil {
		t.Fatalf("save running: %v", err)
	}

	// Create completed saga
	completed := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusCompleted,
		CurrentStep: 2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Save(ctx, completed); err != nil {
		t.Fatalf("save completed: %v", err)
	}

	// Create compensating saga
	compensating := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusCompensating,
		CurrentStep: 1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Save(ctx, compensating); err != nil {
		t.Fatalf("save compensating: %v", err)
	}

	states, err := store.LoadAllRunning(ctx)
	if err != nil {
		t.Fatalf("LoadAllRunning: %v", err)
	}

	if len(states) != 2 {
		t.Fatalf("expected 2 running sagas, got %d", len(states))
	}
}

func TestSQLiteSagaStore_Save_Upsert(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestSagaStore(t)
	ctx := context.Background()

	state := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusPending,
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("first save: %v", err)
	}

	state.Status = saga.StatusRunning
	state.CurrentStep = 1
	state.UpdatedAt = time.Now()

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("second save (upsert): %v", err)
	}

	loaded, err := store.Load(ctx, state.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Status != saga.StatusRunning {
		t.Errorf("Status mismatch after upsert: got %q, want %q", loaded.Status, saga.StatusRunning)
	}
	if loaded.CurrentStep != 1 {
		t.Errorf("CurrentStep mismatch after upsert: got %d, want %d", loaded.CurrentStep, 1)
	}
}

func TestSQLiteOutbox_FullCycle(t *testing.T) {
	t.Parallel()

	// Use SQLBackend to get wired store + outbox
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}

	ctx := context.Background()

	// 1. Create and save an event via TransactionalStore (appends to outbox)
	aggID := id.NewAggregateID()
	evt, err := event.New("order.placed", aggID, "Order", 1, []byte(`{"id":"ORD-123"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := backend.TransactionalStore().SaveWithOutbox(ctx, "Order", aggID, []event.Event{evt}, 0); err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	// 2. PollPending — should return the outbox entry
	entries, err := backend.Outbox().PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(entries))
	}

	if len(entries[0].Events) != 1 {
		t.Fatalf("expected 1 event in entry, got %d", len(entries[0].Events))
	}

	if entries[0].Events[0].Type() != "order.placed" {
		t.Errorf("event type mismatch: got %q, want %q", entries[0].Events[0].Type(), "order.placed")
	}

	// 3. Publish the event (simulated — just verify the entry is valid)
	publishedEvent := entries[0].Events[0]
	if publishedEvent.AggregateID() != aggID {
		t.Errorf("aggregate ID mismatch: got %v, want %v", publishedEvent.AggregateID(), aggID)
	}

	// 4. Ack the outbox entry
	ackIDs := []event.OutboxID{entries[0].ID}
	if err := backend.Outbox().Ack(ctx, ackIDs); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	// 5. PollPending again — should be empty
	entries, err = backend.Outbox().PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("PollPending after ack: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 outbox entries after ack, got %d", len(entries))
	}
}
