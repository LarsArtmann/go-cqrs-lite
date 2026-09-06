//go:build integration

package storage_test

// PostgreSQL integration tests. Run with: go test -tags=integration ./...
//
// Uses testcontainers (postgres:16-alpine) when DATABASE_URL/
// POSTGRES_TEST_DSN is unset. Each test gets its own fresh database.
//
// These tests verify that the SQL stores work correctly against a real PostgreSQL
// instance, not just SQLite. They cover dialect-specific behavior like placeholder
// format ($1, $2 vs ?), timestamp handling, and error classification.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func pgDB(t *testing.T) *sql.DB {
	t.Helper()

	url := pgTestDSN(t)

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

	// Apply schema (fresh per-test databases start empty).
	if err := storage.PostgresInitSchema(ctx, db); err != nil {
		t.Fatalf("PostgresInitSchema: %v", err)
	}

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
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("User", streamID)

	// Save
	evt, _ := event.NewEvent("user.created", streamID, "User", event.Version(1),
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
	evt2, _ := event.NewEvent("user.updated", streamID, "User", event.Version(2),
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
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)

	// Event
	evt, _ := event.NewEvent("test.event", streamID, "Test", event.Version(1), []byte(`{}`))
	_ = eventStore.Save(ctx, ref, []event.Event{evt}, event.Version(0))

	// Snapshot
	_ = snapStore.Save(ctx, snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Test",
		Version:    event.Version(1),
		State:      []byte(`{}`),
		CreatedAt:  time.Now(),
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

func TestPostgresSnapshotColumnMigration(t *testing.T) {
	url := pgTestDSN(t)
	ctx := context.Background()

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Simulate a pre-v5 database: old snapshots table carrying data.
	const legacyDDL = `CREATE TABLE snapshots (
		aggregate_type  VARCHAR(255) NOT NULL,
		aggregate_id    TEXT NOT NULL,
		version         INTEGER NOT NULL,
		state           JSONB NOT NULL,
		created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		PRIMARY KEY (aggregate_type, aggregate_id)
	)`
	if _, err := db.ExecContext(ctx, legacyDDL); err != nil {
		t.Fatalf("create legacy snapshots table: %v", err)
	}

	streamID := id.NewStreamID()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO snapshots (aggregate_type, aggregate_id, version, state, created_at)
		 VALUES ('User', $1, 7, '{"k":"v"}', NOW())`, streamID.String(),
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// v5 boot path: CREATE TABLE IF NOT EXISTS no-ops, the embedded
	// migration renames the columns in place.
	if err := storage.PostgresInitSchema(ctx, db); err != nil {
		t.Fatalf("PostgresInitSchema over legacy table: %v", err)
	}

	store, err := storage.NewSQLSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLSnapshotStore: %v", err)
	}

	got, err := store.Load(ctx, id.NewStreamRef("User", streamID))
	if err != nil {
		t.Fatalf("Load migrated row: %v", err)
	}

	if got.StreamID != streamID || got.StreamType != "User" || got.Version.Int() != 7 {
		t.Errorf("identity mismatch: got %s/%s v%d", got.StreamType, got.StreamID, got.Version.Int())
	}
	if string(got.State) != `{"k":"v"}` {
		t.Errorf("state mismatch: %s", got.State)
	}
}
