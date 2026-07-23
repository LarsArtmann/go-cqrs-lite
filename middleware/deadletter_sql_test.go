package middleware

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func newSQLDeadLetterStore(t *testing.T) *SQLDeadLetterStore {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLDeadLetterStore(db, "sqlite")
	if err != nil {
		t.Fatalf("new sql dead letter store: %v", err)
	}

	return store
}

func TestSQLDeadLetterStore_HandleAndEntries(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	store.Handle(ctx, DeadLetterEntry{
		Kind:     "event",
		Type:     "user.created",
		Error:    errors.New("boom"),
		Attempts: 3,
		FailedAt: now,
	})

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count=1, got %d", count)
	}

	entries, err := store.Entries(ctx)
	if err != nil {
		t.Fatalf("entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]

	if e.Kind != "event" {
		t.Errorf("Kind = %q, want %q", e.Kind, "event")
	}

	if e.Type != "user.created" {
		t.Errorf("Type = %q, want %q", e.Type, "user.created")
	}

	if e.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", e.Attempts)
	}

	if e.Error == nil || e.Error.Error() != "boom" {
		t.Errorf("Error = %v, want %q", e.Error, "boom")
	}

	if e.FailedAt.IsZero() {
		t.Error("FailedAt should not be zero")
	}
}

func TestSQLDeadLetterStore_WithAggregateID(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)
	ctx := context.Background()

	aggID := id.NewStreamID()

	store.Handle(ctx, DeadLetterEntry{
		Kind:     "command",
		Type:     "user.create",
		StreamID: aggID,
		Error:    errors.New("timeout"),
		Attempts: 5,
		FailedAt: time.Now(),
	})

	entries, err := store.Entries(ctx)
	if err != nil {
		t.Fatalf("entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].StreamID != aggID {
		t.Errorf("StreamID = %v, want %v", entries[0].StreamID, aggID)
	}

	if entries[0].Kind != "command" {
		t.Errorf("Kind = %q, want %q", entries[0].Kind, "command")
	}
}

func TestSQLDeadLetterStore_Count(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)
	ctx := context.Background()

	for range 3 {
		store.Handle(ctx, DeadLetterEntry{Kind: "event", Type: "test"})
	}

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 3 {
		t.Fatalf("expected count=3, got %d", count)
	}
}

func TestSQLDeadLetterStore_Clear(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)
	ctx := context.Background()

	store.Handle(ctx, DeadLetterEntry{Kind: "event", Type: "a"})
	store.Handle(ctx, DeadLetterEntry{Kind: "event", Type: "b"})

	count, _ := store.Count(ctx)
	if count != 2 {
		t.Fatalf("expected count=2 before clear, got %d", count)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}

	count, _ = store.Count(ctx)
	if count != 0 {
		t.Fatalf("expected count=0 after clear, got %d", count)
	}
}

func TestSQLDeadLetterStore_PreservesEntriesAcrossInstances(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"_persist?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	store1, err := NewSQLDeadLetterStore(db, "sqlite")
	if err != nil {
		t.Fatalf("new store1: %v", err)
	}

	store1.Handle(ctx, DeadLetterEntry{
		Kind:     "event",
		Type:     "test.persist",
		Error:    errors.New("persisted error"),
		Attempts: 2,
		FailedAt: time.Now(),
	})

	store2, err := NewSQLDeadLetterStore(db, "sqlite")
	if err != nil {
		t.Fatalf("new store2: %v", err)
	}

	entries, err := store2.Entries(ctx)
	if err != nil {
		t.Fatalf("entries from store2: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry from new store instance, got %d", len(entries))
	}

	if entries[0].Type != "test.persist" {
		t.Errorf("Type = %q, want %q", entries[0].Type, "test.persist")
	}

	if entries[0].Error == nil || entries[0].Error.Error() != "persisted error" {
		t.Errorf("Error = %v, want %q", entries[0].Error, "persisted error")
	}
}

func TestSQLDeadLetterStore_IntegrationWithRetry(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)
	ctx := context.Background()

	config := retryConfigFast()
	config.OnDeadLetter = store.Handle

	mw := EventRetry(config)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return errors.New("permanent failure")
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("new test event: %v", err)
	}

	_ = handler(ctx, evt)

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 dead-letter entry after retry exhaustion, got %d", count)
	}

	entries, err := store.Entries(ctx)
	if err != nil {
		t.Fatalf("entries: %v", err)
	}

	if entries[0].Kind != "event" {
		t.Errorf("Kind = %q, want %q", entries[0].Kind, "event")
	}

	if entries[0].Attempts != config.MaxAttempts {
		t.Errorf("Attempts = %d, want %d", entries[0].Attempts, config.MaxAttempts)
	}
}
