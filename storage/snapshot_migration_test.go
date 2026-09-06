package storage

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func newLegacySnapshotsDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const legacyDDL = `CREATE TABLE snapshots (
		aggregate_type  TEXT NOT NULL,
		aggregate_id    TEXT NOT NULL,
		version         INTEGER NOT NULL,
		state           BLOB NOT NULL,
		created_at      TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (aggregate_type, aggregate_id)
	)`
	if _, err := db.Exec(legacyDDL); err != nil {
		t.Fatalf("create legacy snapshots table: %v", err)
	}

	return db
}

func snapshotColumnNames(t *testing.T, db *sql.DB) []string {
	t.Helper()

	names, err := probeSQLiteColumns(context.Background(), db, sqlpkg.TableSnapshots)
	if err != nil {
		t.Fatalf("probe columns: %v", err)
	}

	return names
}

func assertHasColumns(t *testing.T, names []string, want []string, should bool, label string) {
	t.Helper()

	for _, col := range want {
		has := slices.Contains(names, col)
		if should != has {
			t.Errorf("%s: column %q present=%v", label, col, has)
		}
	}
}

func TestMigrateSnapshotColumnsToStream_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newLegacySnapshotsDB(t)

	const insertLegacy = `INSERT INTO snapshots
		(aggregate_type, aggregate_id, version, state, created_at)
		VALUES ('User', ?, 7, ?, '2026-01-02T03:04:05.999Z')`
	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	if _, err := db.Exec(insertLegacy, streamID.String(), []byte("state-bytes")); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.SQLiteDialect{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	names := snapshotColumnNames(t, db)
	assertHasColumns(t, names, []string{"stream_type", "stream_id"}, true, "after migrate")
	assertHasColumns(t, names, []string{"aggregate_type", "aggregate_id"}, false, "after migrate")

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.SQLiteDialect{}); err != nil {
		t.Fatalf("re-migrate (idempotency): %v", err)
	}

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("new snapshot store: %v", err)
	}

	got, err := store.Load(ctx, id.NewStreamRef("User", streamID))
	if err != nil {
		t.Fatalf("load migrated row: %v", err)
	}

	if got.StreamID != streamID || got.StreamType != "User" || got.Version.Int() != 7 {
		t.Errorf(
			"identity mismatch: got %s/%s v%d",
			got.StreamType,
			got.StreamID,
			got.Version.Int(),
		)
	}
	if string(got.State) != "state-bytes" {
		t.Errorf("state mismatch: %q", got.State)
	}
	wantAt := time.Date(2026, 1, 2, 3, 4, 5, 999000000, time.UTC)
	if !got.CreatedAt.Equal(wantAt) {
		t.Errorf("created-at mismatch: got %v want %v", got.CreatedAt, wantAt)
	}
}

func TestMigrateSnapshotColumnsToStream_NoOpWhenFreshOrMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fresh, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer fresh.Close()

	if err := SQLiteInitSchema(ctx, fresh); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	assertHasColumns(t, snapshotColumnNames(t, fresh),
		[]string{"stream_type", "stream_id"}, true, "fresh schema")
	assertHasColumns(t, snapshotColumnNames(t, fresh),
		[]string{"aggregate_type", "aggregate_id"}, false, "fresh schema")

	missing, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer missing.Close()

	if err := MigrateSnapshotColumnsToStream(ctx, missing, sqlpkg.SQLiteDialect{}); err != nil {
		t.Errorf("migrate on missing table must be a no-op, got %v", err)
	}
}
