package projectionhost_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func openSQLiteDLQ(t *testing.T) (*sql.DB, *projectionhost.SQLiteDeadLetterStore) {
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

	return db, store
}

// deadLetterStores pairs the two stores so contract tests pin them to the
// same behavior.
func deadLetterStores(t *testing.T) map[string]projectionhost.DeadLetterStore {
	t.Helper()

	_, sqlite := openSQLiteDLQ(t)

	return map[string]projectionhost.DeadLetterStore{
		"memory": projectionhost.NewMemoryDeadLetterStore(),
		"sqlite": sqlite,
	}
}

// TestDeadLetterStore_Delete_Scopes pins the interface contract: an empty
// projection name deletes by event ID across all projections in EVERY store.
// The SQLite store used to match nothing for that call while the memory
// store deleted globally.
func TestDeadLetterStore_Delete_Scopes(t *testing.T) {
	t.Parallel()

	for name, store := range deadLetterStores(t) {
		ctx := context.Background()

		entry := makeDLQEntry(t, "orders-a")
		for _, proj := range []string{"orders-a", "orders-b"} {
			entry.ProjectionName = proj
			if err := store.Store(ctx, entry); err != nil {
				t.Fatalf("%s: Store %s: %v", name, proj, err)
			}
		}

		other := entry
		other.EventID = id.NewEventID().String()
		other.ProjectionName = "orders-c"
		if err := store.Store(ctx, other); err != nil {
			t.Fatalf("%s: Store other: %v", name, err)
		}

		if err := store.Delete(ctx, "orders-a", entry.EventID); err != nil {
			t.Fatalf("%s: Delete one projection: %v", name, err)
		}

		for proj, want := range map[string]int{"orders-b": 1, "orders-c": 1} {
			if got := countDeadLetters(t, store, proj); got != want {
				t.Errorf("%s: after scoped delete %s has %d entries, want %d", name, proj, got, want)
			}
		}

		if err := store.Delete(ctx, "", entry.EventID); err != nil {
			t.Fatalf("%s: Delete all projections: %v", name, err)
		}

		for proj, want := range map[string]int{"orders-b": 0, "orders-c": 1} {
			if got := countDeadLetters(t, store, proj); got != want {
				t.Errorf("%s: after empty-name delete %s has %d entries, want %d", name, proj, got, want)
			}
		}
	}
}

func countDeadLetters(t *testing.T, store projectionhost.DeadLetterStore, projection string) int {
	t.Helper()

	entries, err := store.List(context.Background(), projection)
	if err != nil {
		t.Fatalf("List %q: %v", projection, err)
	}

	return len(entries)
}

// TestSQLiteDeadLetterStore_CorruptIdentifiersAreSkipped pins that a row
// whose event_id or failed_at cannot be parsed is reported as corrupt and
// skipped: it used to come back with a freshly invented event ID (a phantom
// event for replay) or a zero timestamp (deleted by any PurgeBefore).
func TestSQLiteDeadLetterStore_CorruptIdentifiersAreSkipped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, store := openSQLiteDLQ(t)
	healthy := makeDLQEntry(t, "users-proj")
	if err := store.Store(ctx, healthy); err != nil {
		t.Fatalf("Store healthy: %v", err)
	}

	rows := []struct{ eventID, failedAt string }{
		{"not-a-ulid", time.Now().UTC().Format(time.RFC3339Nano)},
		{id.NewEventID().String(), "not-a-timestamp"},
	}

	for i, row := range rows {
		_, err := db.ExecContext(ctx, `INSERT INTO projection_dead_letters
			(projection_name, event_id, event_type, aggregate_type, aggregate_id,
			 version, schema_version, payload, payload_encoding, metadata, occurred_at,
			 error_text, error_code, error_family, failed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			"users-proj", row.eventID, "test.poison", "Test",
			id.NewStreamID().String(), 1, 1, []byte(`{"bad":true}`), "json", "{}", "",
			"boom", "test.poison", "rejection", row.failedAt)
		if err != nil {
			t.Fatalf("insert corrupt row %d: %v", i, err)
		}
	}

	entries, err := store.List(ctx, "users-proj")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(entries) != 1 || entries[0].EventID != healthy.EventID {
		t.Fatalf("List must return only the healthy row, got %+v", entries)
	}

	if got := store.SkippedCount(); got != 2 {
		t.Fatalf("SkippedCount = %d, want 2", got)
	}
}
