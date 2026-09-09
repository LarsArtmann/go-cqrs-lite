package storage

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// createSnapshotsWithColumns builds a snapshots table with exactly the given
// identity columns (plus the invariant payload columns), for mixed-state and
// legacy-subset fixtures.
func createSnapshotsWithColumns(t *testing.T, db *sql.DB, identityCols string) {
	t.Helper()

	ddl := "CREATE TABLE snapshots (" + identityCols + "," + `
		version    INTEGER NOT NULL,
		state      BLOB NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')))`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("create snapshots table (%s): %v", identityCols, err)
	}
}

// TestMigrateSnapshotColumns_MixedStateRejected pins the corruption guard: a
// table carrying BOTH spellings (someone half-migrated by hand, or a crashed
// migration left half-renamed) must fail loudly with the mixed-state
// Corruption code instead of renaming anything.
func TestMigrateSnapshotColumns_MixedStateRejected(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	createSnapshotsWithColumns(t, db,
		"aggregate_type TEXT NOT NULL, aggregate_id TEXT NOT NULL, "+
			"stream_type TEXT NOT NULL, stream_id TEXT NOT NULL")

	err = MigrateSnapshotColumnsToStream(context.Background(), db, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("mixed-state table must be rejected")
	}

	if errorfamily.Classify(err) != errorfamily.Corruption {
		t.Fatalf("expected Corruption family, got %v (%v)", errorfamily.Classify(err), err)
	}

	if famErr, ok := errors.AsType[*errorfamily.Error](err); !ok ||
		famErr.Code() != "storage.snapshot_column_mixed" {
		t.Fatalf("expected storage.snapshot_column_mixed code, got %v", err)
	}
}

// TestMigrateSnapshotColumns_HalfMigratedStateRejected pins the observable of
// a crash BETWEEN the two ALTERs: stream_type exists, aggregate_id remains.
// The next boot must hit the same mixed-state guard — never silently rename
// the survivor — so an operator reconciles once instead of chasing drift.
func TestMigrateSnapshotColumns_HalfMigratedStateRejected(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	createSnapshotsWithColumns(t, db,
		"stream_type TEXT NOT NULL, aggregate_id TEXT NOT NULL")

	err = MigrateSnapshotColumnsToStream(context.Background(), db, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("half-migrated table must be rejected")
	}

	if famErr, ok := errors.AsType[*errorfamily.Error](err); !ok ||
		famErr.Code() != "storage.snapshot_column_mixed" {
		t.Fatalf("expected storage.snapshot_column_mixed code, got %v", err)
	}
}

// TestMigrateSnapshotColumns_LegacySubsetRenamesWhatExists pins the subset
// path: a table carrying only ONE legacy column (partial pre-v5 schema)
// renames exactly that column and leaves the other spellings untouched.
func TestMigrateSnapshotColumns_LegacySubsetRenamesWhatExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	createSnapshotsWithColumns(t, db, "aggregate_id TEXT NOT NULL")

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.SQLiteDialect{}); err != nil {
		t.Fatalf("migrate legacy subset: %v", err)
	}

	names := snapshotColumnNames(t, db)
	if !slices.Contains(names, "stream_id") {
		t.Fatalf("aggregate_id should have been renamed to stream_id, columns: %v", names)
	}

	if slices.Contains(names, "aggregate_id") || slices.Contains(names, "aggregate_type") {
		t.Fatalf("legacy columns must be gone, columns: %v", names)
	}

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.SQLiteDialect{}); err != nil {
		t.Fatalf("re-migrate after subset rename (idempotency): %v", err)
	}
}

// TestMigrateSnapshotColumns_ConcurrentInitIsSafe pins the concurrent-boot
// contract: N processes calling the migration on the same legacy table at
// once all observe success (or the table is already fully migrated), and the
// final schema is fully renamed — never half, never erroring after another
// runner won the race.
func TestMigrateSnapshotColumns_ConcurrentInitIsSafe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newLegacySnapshotsDB(t)

	const runners = 8

	start := make(chan struct{})
	errs := make([]error, runners)

	var wg sync.WaitGroup

	for i := range runners {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			errs[i] = MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.SQLiteDialect{})
		}()
	}

	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("runner %d failed: %v", i, err)
		}
	}

	names := snapshotColumnNames(t, db)
	assertHasColumns(t, names, []string{"stream_type", "stream_id"}, true, "after concurrent migrate")
	assertHasColumns(t, names, []string{"aggregate_type", "aggregate_id"}, false, "after concurrent migrate")
}
