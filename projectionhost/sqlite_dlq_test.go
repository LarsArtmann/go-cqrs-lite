package projectionhost_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
)

func newSQLiteDLQ(t *testing.T) *projectionhost.SQLiteDeadLetterStore {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	store, err := projectionhost.NewSQLiteDeadLetterStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteDeadLetterStore: %v", err)
	}

	return store
}

func makeDLQEntry(t *testing.T, projName string) projectionhost.DeadLetterEntry {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		"test.poison",
		aggID,
		"Test",
		event.Version(1),
		[]byte(`{"bad":true}`),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return projectionhost.DeadLetterEntry{
		ProjectionName: projName,
		EventID:        evt.ID().String(),
		EventType:      string(evt.Type()),
		AggregateID:    aggID.String(),
		Event:          evt,
		Error:          "handler panicked: nil pointer",
		ErrorCode:      "test.panic",
		ErrorFamily:    "infrastructure",
		FailedAt:       time.Now().UTC(),
	}
}

func TestSQLiteDeadLetterStore_NilDB(t *testing.T) {
	t.Parallel()

	store, err := projectionhost.NewSQLiteDeadLetterStore(nil)
	if store != nil {
		t.Fatal("expected nil store")
	}

	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestSQLiteDeadLetterStore_StoreAndList(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	entry := makeDLQEntry(t, "users-proj")

	if err := store.Store(ctx, entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	listed, err := store.List(ctx, "users-proj")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(listed))
	}

	got := listed[0]
	if got.ProjectionName != "users-proj" {
		t.Errorf("ProjectionName = %q, want users-proj", got.ProjectionName)
	}

	if got.EventID != entry.EventID {
		t.Errorf("EventID = %q, want %q", got.EventID, entry.EventID)
	}

	if got.EventType != "test.poison" {
		t.Errorf("EventType = %q, want test.poison", got.EventType)
	}

	if got.Error != "handler panicked: nil pointer" {
		t.Errorf("Error = %q", got.Error)
	}

	if got.Event == nil {
		t.Fatal("reconstructed event is nil")
	}

	if string(got.Event.Payload()) != `{"bad":true}` {
		t.Errorf("payload = %q", string(got.Event.Payload()))
	}
}

func TestSQLiteDeadLetterStore_ListAll(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	e1 := makeDLQEntry(t, "users-proj")
	e2 := makeDLQEntry(t, "orders-proj")

	if err := store.Store(ctx, e1); err != nil {
		t.Fatalf("Store e1: %v", err)
	}

	if err := store.Store(ctx, e2); err != nil {
		t.Fatalf("Store e2: %v", err)
	}

	all, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
}

func TestSQLiteDeadLetterStore_ListByProjection(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	e1 := makeDLQEntry(t, "users-proj")
	e2 := makeDLQEntry(t, "orders-proj")

	_ = store.Store(ctx, e1)
	_ = store.Store(ctx, e2)

	users, err := store.List(ctx, "users-proj")
	if err != nil {
		t.Fatalf("List users-proj: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 entry for users-proj, got %d", len(users))
	}

	if users[0].ProjectionName != "users-proj" {
		t.Errorf("ProjectionName = %q", users[0].ProjectionName)
	}
}

func TestSQLiteDeadLetterStore_Delete(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	entry := makeDLQEntry(t, "users-proj")
	_ = store.Store(ctx, entry)

	if err := store.Delete(ctx, "users-proj", entry.EventID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	listed, _ := store.List(ctx, "users-proj")
	if len(listed) != 0 {
		t.Fatalf("expected 0 entries after delete, got %d", len(listed))
	}
}

func TestSQLiteDeadLetterStore_Purge(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	e1 := makeDLQEntry(t, "users-proj")
	e2 := makeDLQEntry(t, "users-proj")
	e3 := makeDLQEntry(t, "orders-proj")

	_ = store.Store(ctx, e1)
	_ = store.Store(ctx, e2)
	_ = store.Store(ctx, e3)

	if err := store.Purge(ctx, "users-proj"); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	users, _ := store.List(ctx, "users-proj")
	if len(users) != 0 {
		t.Fatalf("expected 0 users entries after purge, got %d", len(users))
	}

	orders, _ := store.List(ctx, "orders-proj")
	if len(orders) != 1 {
		t.Fatalf("orders should be untouched, got %d", len(orders))
	}
}

func TestSQLiteDeadLetterStore_PurgeAll(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	_ = store.Store(ctx, makeDLQEntry(t, "a"))
	_ = store.Store(ctx, makeDLQEntry(t, "b"))

	if err := store.Purge(ctx, ""); err != nil {
		t.Fatalf("Purge all: %v", err)
	}

	all, _ := store.List(ctx, "")
	if len(all) != 0 {
		t.Fatalf("expected 0 entries after purge all, got %d", len(all))
	}
}

func TestSQLiteDeadLetterStore_ReplaceOnDuplicate(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	entry := makeDLQEntry(t, "users-proj")
	entry.Error = "first error"

	if err := store.Store(ctx, entry); err != nil {
		t.Fatalf("Store first: %v", err)
	}

	entry.Error = "second error"

	if err := store.Store(ctx, entry); err != nil {
		t.Fatalf("Store second: %v", err)
	}

	listed, _ := store.List(ctx, "users-proj")
	if len(listed) != 1 {
		t.Fatalf("expected 1 entry (replaced), got %d", len(listed))
	}

	if listed[0].Error != "second error" {
		t.Errorf("Error = %q, want second error", listed[0].Error)
	}
}

func TestSQLiteDeadLetterStore_PreservesEventFields(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		"order.created", aggID, "Order", event.Version(5),
		[]byte(`{"amount":42}`),
		event.WithSchemaVersion(3),
	)

	entry := projectionhost.DeadLetterEntry{
		ProjectionName: "orders",
		EventID:        evt.ID().String(),
		EventType:      "order.created",
		AggregateID:    aggID.String(),
		Event:          evt,
		Error:          "handler timeout",
		ErrorCode:      "test.timeout",
		ErrorFamily:    "transient",
		FailedAt:       time.Now().UTC(),
	}

	if err := store.Store(ctx, entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	listed, err := store.List(ctx, "orders")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(listed))
	}

	got := listed[0]

	if got.Event == nil {
		t.Fatal("event not reconstructed")
	}

	if got.Event.Version() != 5 {
		t.Errorf("Version = %d, want 5", got.Event.Version())
	}

	if got.Event.SchemaVersion().Int() != 3 {
		t.Errorf("SchemaVersion = %d, want 3", got.Event.SchemaVersion().Int())
	}

	if got.Event.AggregateType() != "Order" {
		t.Errorf("AggregateType = %q", got.Event.AggregateType())
	}
}
