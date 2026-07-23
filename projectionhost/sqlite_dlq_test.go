package projectionhost_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func newSQLiteDLQ(t *testing.T) *projectionhost.SQLiteDeadLetterStore {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	store, err := projectionhost.NewSQLiteDeadLetterStore(t.Context(), db)
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
		StreamID:    aggID.String(),
		Event:          evt,
		Error:          "handler panicked: nil pointer",
		ErrorCode:      "test.panic",
		ErrorFamily:    "infrastructure",
		FailedAt:       time.Now().UTC(),
	}
}

func TestSQLiteDeadLetterStore_NilDB(t *testing.T) {
	t.Parallel()

	store, err := projectionhost.NewSQLiteDeadLetterStore(t.Context(), nil)
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
		StreamID:    aggID.String(),
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

	if got.Event.StreamType() != "Order" {
		t.Errorf("StreamType = %q", got.Event.StreamType())
	}
}

func TestSQLiteDeadLetterStore_Count(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	_ = store.Store(ctx, makeDLQEntry(t, "a"))
	_ = store.Store(ctx, makeDLQEntry(t, "b"))
	_ = store.Store(ctx, makeDLQEntry(t, "c"))

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count after store: %v", err)
	}

	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}
}

func TestSQLiteDeadLetterStore_ListPaged(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		entry := makeDLQEntry(t, "users-proj")
		entry.FailedAt = time.Now().UTC().Add(time.Duration(i) * time.Minute)
		_ = store.Store(ctx, entry)
	}

	page1, err := store.ListPaged(ctx, "users-proj", 0, 3)
	if err != nil {
		t.Fatalf("ListPaged page1: %v", err)
	}

	if len(page1) != 3 {
		t.Fatalf("expected 3 entries on page1, got %d", len(page1))
	}

	page2, err := store.ListPaged(ctx, "users-proj", 3, 3)
	if err != nil {
		t.Fatalf("ListPaged page2: %v", err)
	}

	if len(page2) != 3 {
		t.Fatalf("expected 3 entries on page2, got %d", len(page2))
	}

	lastPage, err := store.ListPaged(ctx, "users-proj", 9, 3)
	if err != nil {
		t.Fatalf("ListPaged lastPage: %v", err)
	}

	if len(lastPage) != 1 {
		t.Fatalf("expected 1 entry on last page, got %d", len(lastPage))
	}

	empty, err := store.ListPaged(ctx, "users-proj", 100, 3)
	if err != nil {
		t.Fatalf("ListPaged beyond: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected 0 entries beyond data, got %d", len(empty))
	}
}

func TestSQLiteDeadLetterStore_ListPaged_AllProjections(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	_ = store.Store(ctx, makeDLQEntry(t, "a"))
	_ = store.Store(ctx, makeDLQEntry(t, "b"))

	page, err := store.ListPaged(ctx, "", 0, 10)
	if err != nil {
		t.Fatalf("ListPaged all: %v", err)
	}

	if len(page) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(page))
	}
}

func TestSQLiteDeadLetterStore_PurgeBefore(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	now := time.Now().UTC()

	old := makeDLQEntry(t, "users-proj")
	old.FailedAt = now.Add(-2 * time.Hour)
	_ = store.Store(ctx, old)

	recent := makeDLQEntry(t, "users-proj")
	recent.FailedAt = now.Add(-10 * time.Minute)
	_ = store.Store(ctx, recent)

	deleted, err := store.PurgeBefore(ctx, now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("PurgeBefore: %v", err)
	}

	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	remaining, _ := store.List(ctx, "users-proj")
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining entry, got %d", len(remaining))
	}
}

func TestSQLiteDeadLetterStore_PurgeBefore_None(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	_ = store.Store(ctx, makeDLQEntry(t, "users-proj"))

	deleted, err := store.PurgeBefore(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("PurgeBefore: %v", err)
	}

	if deleted != 0 {
		t.Fatalf("expected 0 deleted, got %d", deleted)
	}
}

func TestSQLiteDeadLetterStore_DeadLetterStoreAdmin(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)

	var _ projectionhost.DeadLetterStoreAdmin = store
}

func TestSQLiteDeadLetterStore_Stress_10k(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	const n = 10_000

	for i := 0; i < n; i++ {
		entry := makeDLQEntry(t, fmt.Sprintf("proj-%d", i%5))
		_ = store.Store(ctx, entry)
	}

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if count != n {
		t.Fatalf("expected %d entries, got %d", n, count)
	}

	start := time.Now()

	page, err := store.ListPaged(ctx, "proj-0", 0, 100)
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}

	elapsed := time.Since(start)

	if len(page) != 100 {
		t.Fatalf("expected 100 entries, got %d", len(page))
	}

	if elapsed > 100*time.Millisecond {
		t.Logf("WARNING: ListPaged took %v (expected <100ms)", elapsed)
	}

	deleted, err := store.PurgeBefore(ctx, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("PurgeBefore future: %v", err)
	}

	if deleted != n {
		t.Fatalf("expected %d deleted, got %d", n, deleted)
	}
}

func TestSQLiteDeadLetterStore_ConcurrentStore(t *testing.T) {
	t.Parallel()

	store := newSQLiteDLQ(t)
	ctx := context.Background()

	const goroutines = 20

	const perGoroutine = 50

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(projName string) {
			defer wg.Done()

			for i := 0; i < perGoroutine; i++ {
				entry := makeDLQEntry(t, projName)
				_ = store.Store(ctx, entry)
			}
		}(fmt.Sprintf("proj-%d", g%3))
	}

	wg.Wait()

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count after concurrent stores: %v", err)
	}

	expected := int64(goroutines * perGoroutine)

	if count != expected {
		t.Fatalf("expected %d entries after concurrent writes, got %d", expected, count)
	}
}

func TestSQLiteDeadLetterStore_CorruptPayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	store, err := projectionhost.NewSQLiteDeadLetterStore(t.Context(), db)
	if err != nil {
		t.Fatalf("NewSQLiteDeadLetterStore: %v", err)
	}

	_, err = db.ExecContext(
		ctx, `INSERT INTO projection_dead_letters
        (projection_name, event_id, event_type, aggregate_type, aggregate_id,
         version, schema_version, payload, payload_encoding, metadata, occurred_at,
         error_text, error_code, error_family, failed_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"corrupt-proj",
		"evt-corrupt",
		"test.corrupt",
		"Test",
		"01H8XQDVQNNJ1TF4D2JEEQX5S2",
		1,
		1,
		[]byte("\x00\x01\x02not-valid-json\x00"),
		"json",
		"{\"corrupt\": true",
		time.Now().UTC().Format(time.RFC3339Nano),
		"corruption error",
		"test.corrupt",
		"corruption",
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("insert corrupt row: %v", err)
	}

	entries, err := store.List(ctx, "corrupt-proj")
	if err != nil {
		t.Logf("List returned error on corrupt payload (acceptable): %v", err)

		return
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.EventID != "evt-corrupt" {
		t.Errorf("EventID = %q, want evt-corrupt", got.EventID)
	}

	if got.Error != "corruption error" {
		t.Errorf("Error = %q, want corruption error", got.Error)
	}
}
