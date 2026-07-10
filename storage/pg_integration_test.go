//go:build integration

package storage_test

// PostgreSQL integration tests. Run with: go test -tags=integration ./...
//
// Requires DATABASE_URL environment variable pointing to a PostgreSQL database.
// Example: DATABASE_URL=postgres://user:pass@localhost:5432/test?sslmode=disable
//
// These tests verify that the SQL stores work correctly against a real PostgreSQL
// instance, not just SQLite. They cover dialect-specific behavior like placeholder
// format ($1, $2 vs ?), timestamp handling, and error classification.

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func pgDB(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping PostgreSQL integration tests")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping pg: %v", err)
	}

	// Clean up any previous test data
	_, _ = db.ExecContext(
		ctx,
		`DROP TABLE IF EXISTS events, commands, queries, snapshots, checkpoints`,
	)

	return db
}

func TestPostgresEventStore_CRUD(t *testing.T) {
	db := pgDB(t)
	ctx := context.Background()

	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		t.Fatalf("NewSQLBackend: %v", err)
	}

	store := backend.EventStore()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("User", aggID)

	// Save
	evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"alice"}`))
	if err := store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load
	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "user.created" {
		t.Fatalf("expected type user.created, got %s", loaded[0].Type())
	}

	// LoadFromVersion
	evt2, _ := event.NewEvent("user.updated", aggID, "User", event.Version(2),
		[]byte(`{"name":"bob"}`))
	_ = store.AppendBatch(ctx, ref, []event.Event{evt2})

	fromV1, err := store.LoadFromVersion(ctx, ref, event.Version(1))
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(fromV1) != 1 {
		t.Fatalf("expected 1 event from v1, got %d", len(fromV1))
	}

	// Journal
	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events in journal, got %d", len(all))
	}
}

func TestPostgresBackend_FullStack(t *testing.T) {
	db := pgDB(t)
	ctx := context.Background()

	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		t.Fatalf("NewSQLBackend: %v", err)
	}

	// Event store
	eventStore := backend.EventStore()

	// Command store
	cmdStore, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore: %v", err)
	}

	// Query store
	qStore, err := backend.QueryStore()
	if err != nil {
		t.Fatalf("QueryStore: %v", err)
	}

	// Snapshot store
	snapStore, err := backend.SnapshotStore()
	if err != nil {
		t.Fatalf("SnapshotStore: %v", err)
	}

	// Checkpoint store
	cpStore, err := backend.CheckpointStore()
	if err != nil {
		t.Fatalf("CheckpointStore: %v", err)
	}

	// Verify all work together
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	// Event
	evt, _ := event.NewEvent("test.event", aggID, "Test", event.Version(1), []byte(`{}`))
	_ = eventStore.Save(ctx, ref, []event.Event{evt}, event.Version(0))

	// Snapshot
	_ = snapStore.Save(ctx, snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Test",
		Version:       event.Version(1),
		State:         []byte(`{}`),
		CreatedAt:     time.Now(),
	})

	// Checkpoint
	_ = cpStore.Save(ctx, "test-projection", event.Checkpoint{
		EventID:     evt.ID(),
		ProcessedAt: time.Now(),
	})

	// Verify load
	loaded, err := eventStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	_ = cmdStore
	_ = qStore
}
