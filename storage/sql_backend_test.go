package storage

import (
	"context"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func newTestSQLBackend(t *testing.T) *SQLBackend {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}

	return backend
}

func TestNewSQLBackend_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteBackend_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLiteBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestSQLBackend_EventStore(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)

	store := backend.EventStore()
	if store == nil {
		t.Fatal("expected non-nil EventStore")
	}
}

func TestSQLBackend_Outbox(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)

	outbox := backend.Outbox()
	if outbox == nil {
		t.Fatal("expected non-nil Outbox")
	}
}

func TestSQLBackend_TransactionalStore(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)

	tx := backend.TransactionalStore()
	if tx == nil {
		t.Fatal("expected non-nil TransactionalStore")
	}
}

func TestSQLBackend_SagaStore(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)

	sagaStore := backend.SagaStore()
	if sagaStore == nil {
		t.Fatal("expected non-nil SagaStore")
	}
}

func TestSQLBackend_SagaStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)
	ctx := context.Background()

	state := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusRunning,
		CurrentStep: 1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := backend.SagaStore().Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := backend.SagaStore().Load(ctx, state.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != state.ID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, state.ID)
	}
}

func TestSQLBackend_SaveWithOutbox(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt, err := event.New("test.event", aggID, "Test", 1, []byte(`{"data":1}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	tx := backend.TransactionalStore()
	if err := tx.SaveWithOutbox(ctx, "Test", aggID, []event.Event{evt}, 0); err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	// Verify event was saved
	loaded, err := backend.EventStore().Load(ctx, "Test", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "test.event" {
		t.Errorf("event type mismatch: got %q, want %q", loaded[0].Type(), "test.event")
	}

	// Verify outbox entry was created
	entries, err := backend.Outbox().PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(entries))
	}
}

func TestNewSQLBackendWithDialect(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLBackendWithDialect(db, SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLBackendWithDialect: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
	if backend.EventStore() == nil {
		t.Fatal("expected non-nil EventStore")
	}
	if backend.Outbox() == nil {
		t.Fatal("expected non-nil Outbox")
	}
	if backend.TransactionalStore() == nil {
		t.Fatal("expected non-nil TransactionalStore")
	}
	if backend.SagaStore() == nil {
		t.Fatal("expected non-nil SagaStore")
	}
}
