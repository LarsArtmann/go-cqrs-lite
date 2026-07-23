package eventstore

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// TestSQLEventStore_SaveLoadRoundtrip is a smoke test that exercises the
// eventstore package directly (not through the storage aliases), proving the
// sub-package is independently usable and has local test coverage.
func TestSQLEventStore_SaveLoadRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, sqlpkg.SQLiteSchemaEmbed()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)

	evt, err := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Load: got %d events, want 1", len(events))
	}

	if events[0].Type() != "user.created" {
		t.Fatalf("Type: got %q, want %q", events[0].Type(), "user.created")
	}
}

func TestNewSQLiteEventStore_NilDB(t *testing.T) {
	t.Parallel()

	if _, err := NewSQLiteEventStore(nil); err == nil {
		t.Fatal("expected error for nil db")
	}
}
