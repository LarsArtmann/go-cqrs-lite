package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func newShapeTestStore(t *testing.T) *SQLSnapshotStore {
	t.Helper()

	db := newSQLiteTestDB(t)
	if err := SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	return store
}

// TestSQLiteSnapshotStore_StateShapeRoundTrip pins issue #117's fix: the
// stamp a decider writes via WithSnapshotStateVersion must survive the SQL
// store round trip, or the mismatch-discard guard never fires on reload.
func TestSQLiteSnapshotStore_StateShapeRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newShapeTestStore(t)
	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	ref := id.NewStreamRef("User", streamID)

	stamped := snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "User",
		Version:    event.Version(4),
		State:      []byte(`{"name":"John"}`),
		StateShape: "v2",
		CreatedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := store.Save(ctx, stamped); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.StateShape != "v2" {
		t.Fatalf("Load dropped StateShape: got %q, want %q", loaded.StateShape, "v2")
	}

	atVersion, err := store.LoadAtVersion(ctx, ref, event.Version(10))
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}
	if atVersion.StateShape != "v2" {
		t.Fatalf("LoadAtVersion dropped StateShape: got %q, want %q", atVersion.StateShape, "v2")
	}

	// An upsert over the same stream must move the stamp too (shape bumps
	// re-save the same stream identity with the new stamp).
	stamped.StateShape = "v3"
	stamped.Version = event.Version(5)
	if err := store.Save(ctx, stamped); err != nil {
		t.Fatalf("Save v3: %v", err)
	}
	reloaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load after re-save: %v", err)
	}
	if reloaded.StateShape != "v3" || reloaded.Version.Int() != 5 {
		t.Fatalf("upsert lost stamp/version: shape %q, version %d", reloaded.StateShape, reloaded.Version)
	}
}

// TestMigrateSnapshotStateShapeColumn_SQLite pins the boot migration: a
// database whose snapshots table predates the column gets it added, legacy
// rows read back unstamped (absent = accept), and stamped writes land.
func TestMigrateSnapshotStateShapeColumn_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newSQLiteTestDB(t)

	const preStampDDL = `CREATE TABLE snapshots (
		stream_type  TEXT NOT NULL,
		stream_id    TEXT NOT NULL,
		version      INTEGER NOT NULL,
		state        BLOB NOT NULL,
		created_at   TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (stream_type, stream_id)
	)`
	if _, err := db.ExecContext(ctx, preStampDDL); err != nil {
		t.Fatalf("create pre-stamp table: %v", err)
	}

	const insertLegacy = `INSERT INTO snapshots
		(stream_type, stream_id, version, state, created_at)
		VALUES ('User', ?, 7, ?, '2026-01-02T03:04:05.999Z')`
	legacyID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	if _, err := db.ExecContext(ctx, insertLegacy, legacyID.Get(), []byte("state-bytes")); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := SQLiteInitSchema(ctx, db); err != nil {
		t.Fatalf("SQLiteInitSchema over legacy table: %v", err)
	}

	assertHasColumns(
		t,
		snapshotColumnNames(t, db),
		[]string{"state_shape"},
		true,
		"after migration",
	)

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	legacySnap, err := store.Load(ctx, id.NewStreamRef("User", legacyID))
	if err != nil {
		t.Fatalf("Load legacy row: %v", err)
	}
	if legacySnap.StateShape != "" {
		t.Fatalf("legacy row gained a stamp: %q", legacySnap.StateShape)
	}

	stamped := snapshot.Snapshot{
		StreamID:   legacyID,
		StreamType: "User",
		Version:    event.Version(8),
		State:      []byte(`{}`),
		StateShape: "v2",
		CreatedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := store.Save(ctx, stamped); err != nil {
		t.Fatalf("Save stamped: %v", err)
	}
	loaded, err := store.Load(ctx, id.NewStreamRef("User", legacyID))
	if err != nil {
		t.Fatalf("Load stamped: %v", err)
	}
	if loaded.StateShape != "v2" {
		t.Fatalf("stamped row lost its stamp: %q", loaded.StateShape)
	}

	// Idempotent: a second boot is a no-op, not an error.
	if err := MigrateSnapshotStateShapeColumn(ctx, db, sqlpkg.SQLiteDialect{}); err != nil {
		t.Fatalf("second migration run: %v", err)
	}
}

// TestMigrateSnapshotStateShapeColumn_ProbeFailure pins the failure family of
// the column probe: a database that cannot answer the probe surfaces as an
// Infrastructure error carrying storage.snapshot_state_shape_probe, never as a
// raw sql error or a silent nil.
func TestMigrateSnapshotStateShapeColumn_ProbeFailure(t *testing.T) {
	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	err = MigrateSnapshotStateShapeColumn(context.Background(), db, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("expected probe failure on a closed database, got nil")
	}
	if errorfamily.Classify(err) != errorfamily.Infrastructure {
		t.Fatalf("expected Infrastructure family, got %v (%v)", errorfamily.Classify(err), err)
	}
	if !strings.Contains(err.Error(), "storage.snapshot_state_shape_probe") {
		t.Fatalf("error does not name the probe code: %v", err)
	}
}

// TestMigrateSnapshotStateShapeColumn_AddFailure pins the failure family of
// the ALTER: a snapshots relation that cannot take the new column (here: a
// view, which SQLite refuses to ALTER) surfaces as an Infrastructure error
// carrying storage.snapshot_state_shape_add.
func TestMigrateSnapshotStateShapeColumn_AddFailure(t *testing.T) {
	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// A view named snapshots: the column probe succeeds (views report
	// columns), but ALTER TABLE ... ADD COLUMN is refused on views.
	if _, err := db.Exec(`CREATE VIEW snapshots AS SELECT 1 AS aggregate_id`); err != nil {
		t.Fatalf("create snapshots view: %v", err)
	}

	err = MigrateSnapshotStateShapeColumn(context.Background(), db, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("expected ALTER failure on a view, got nil")
	}
	if errorfamily.Classify(err) != errorfamily.Infrastructure {
		t.Fatalf("expected Infrastructure family, got %v (%v)", errorfamily.Classify(err), err)
	}
	if !strings.Contains(err.Error(), "storage.snapshot_state_shape_add") {
		t.Fatalf("error does not name the add code: %v", err)
	}
}
