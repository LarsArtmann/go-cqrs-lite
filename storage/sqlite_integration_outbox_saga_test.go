package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

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

	testhelpers.SaveSagaState(t, ctx, store, state)

	loaded, err := store.Load(ctx, state.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != state.ID {
		testhelpers.AssertEqual(t, loaded.ID, state.ID, "ID")
	}
	if loaded.SagaType != state.SagaType {
		testhelpers.AssertEqual(t, loaded.SagaType, state.SagaType, "SagaType")
	}
	if loaded.Status != state.Status {
		testhelpers.AssertEqual(t, loaded.Status, state.Status, "Status")
	}
	if loaded.CurrentStep != state.CurrentStep {
		testhelpers.AssertEqual(t, loaded.CurrentStep, state.CurrentStep, "CurrentStep")
	}
	if loaded.ErrMsg != state.ErrMsg {
		testhelpers.AssertEqual(t, loaded.ErrMsg, state.ErrMsg, "ErrMsg")
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

	running := testhelpers.NewSagaState("order", saga.StatusRunning, 1)
	testhelpers.SaveSagaState(t, ctx, store, running)

	completed := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusCompleted,
		CurrentStep: 2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	testhelpers.SaveSagaState(t, ctx, store, completed)

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

	testhelpers.SaveSagaState(t, ctx, store, state)

	state.Status = saga.StatusRunning
	state.CurrentStep = 1
	state.UpdatedAt = time.Now()

	testhelpers.SaveSagaState(t, ctx, store, state)

	loaded, err := store.Load(ctx, state.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Status != saga.StatusRunning {
		testhelpers.AssertEqual(t, loaded.Status, saga.StatusRunning, "Status")
	}
	if loaded.CurrentStep != 1 {
		testhelpers.AssertEqual(t, loaded.CurrentStep, 1, "CurrentStep")
	}
}

func TestSQLiteOutbox_FullCycle(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}

	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt, err := event.New("order.placed", aggID, "Order", 1, []byte(`{"id":"ORD-123"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := backend.TransactionalStore().
		SaveWithOutbox(ctx, "Order", aggID, []event.Event{evt}, 0); err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

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
		t.Errorf(
			"event type mismatch: got %q, want %q",
			entries[0].Events[0].Type(),
			"order.placed",
		)
	}

	publishedEvent := entries[0].Events[0]
	if publishedEvent.AggregateID() != aggID {
		t.Errorf("aggregate ID mismatch: got %v, want %v", publishedEvent.AggregateID(), aggID)
	}

	ackIDs := []event.OutboxID{entries[0].ID}
	if err := backend.Outbox().Ack(ctx, ackIDs); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	entries, err = backend.Outbox().PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("PollPending after ack: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 outbox entries after ack, got %d", len(entries))
	}
}
