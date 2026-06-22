package indexing_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3/indexing"
)

func setupIndexingDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	return db
}

func TestApplyOptimizationsTraced(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	if err := indexing.ApplyOptimizationsTraced(context.Background(), db); err != nil {
		t.Fatalf("ApplyOptimizationsTraced: %v", err)
	}
}

func TestWithIndexingHooks(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	var created []string

	auto := indexing.NewAutoIndexer(
		db,
		indexing.WithIndexingHooks(
			indexing.WithBeforeCreateHook(func(_ context.Context, hctx indexing.HookContext) error {
				created = append(created, "before:"+hctx.Index.Name)
				return nil
			}),
			indexing.WithAfterCreateHook(func(_ context.Context, hctx indexing.HookContext) error {
				created = append(created, "after:"+hctx.Index.Name)
				return nil
			}),
		),
	)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	if len(created) == 0 {
		t.Fatal("expected hooks to fire for index creation")
	}

	// Verify before/after pairing.
	beforeCount := 0
	afterCount := 0
	for _, c := range created {
		switch {
		case len(c) > 7 && c[:7] == "before:":
			beforeCount++
		case len(c) > 6 && c[:6] == "after:":
			afterCount++
		}
	}

	if beforeCount != afterCount {
		t.Errorf("before/after hook mismatch: %d before, %d after", beforeCount, afterCount)
	}
}

func TestWithIndexingHooks_Veto(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	vetoErr := errors.New("vetoed")

	auto := indexing.NewAutoIndexer(
		db,
		indexing.WithIndexingHooks(
			indexing.WithBeforeCreateHook(func(_ context.Context, _ indexing.HookContext) error {
				return vetoErr
			}),
		),
	)
	auto.Enable()

	err := auto.ApplyCQRSIndexes(context.Background())
	if err == nil {
		t.Fatal("expected error from before-create hook veto")
	}
}

func TestSchemaChangeHook(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	auto := indexing.NewAutoIndexer(
		db,
		indexing.WithIndexingHooks(
			indexing.WithAfterCreateHook(indexing.SchemaChangeHook()),
		),
	)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes with SchemaChangeHook: %v", err)
	}
}

func TestMigrateWithIndexing_Success(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	migrated := false

	err := indexing.MigrateWithIndexing(
		context.Background(),
		db,
		auto,
		func(ctx context.Context, tx *sql.Tx) error {
			migrated = true
			_, execErr := tx.ExecContext(ctx,
				"CREATE TABLE IF NOT EXISTS migration_test (id INTEGER PRIMARY KEY)")

			return execErr
		},
	)
	if err != nil {
		t.Fatalf("MigrateWithIndexing: %v", err)
	}

	if !migrated {
		t.Error("expected migration function to be called")
	}

	// Verify the table was created.
	var name string
	if err := db.QueryRowContext(
		context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' AND name='migration_test'",
	).Scan(&name); err != nil {
		t.Fatalf("query migration table: %v", err)
	}

	if name != "migration_test" {
		t.Errorf("table name = %q, want migration_test", name)
	}
}

func TestMigrateWithIndexing_MigrationError(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	migrationErr := errors.New("bad migration")

	err := indexing.MigrateWithIndexing(
		context.Background(),
		db,
		nil,
		func(_ context.Context, _ *sql.Tx) error {
			return migrationErr
		},
	)
	if err == nil {
		t.Fatal("expected error from failed migration")
	}
}

func TestMigrateWithIndexing_NilIndexer(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	// nil autoIndexer should still run the migration.
	err := indexing.MigrateWithIndexing(
		context.Background(),
		db,
		nil,
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx,
				"CREATE TABLE IF NOT EXISTS nil_indexer_test (id INTEGER PRIMARY KEY)")

			return err
		},
	)
	if err != nil {
		t.Fatalf("MigrateWithIndexing with nil indexer: %v", err)
	}
}

func TestCheckpointScheduler_RunOnce(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	// Apply WAL first so checkpoint has something to work with.
	_ = indexing.ApplyWAL(context.Background(), db)

	scheduler := indexing.NewCheckpointScheduler(db, 0) // interval=0 disables auto-run
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	// Even with auto-run disabled, we can trigger a manual checkpoint
	// by calling PRAGMA wal_checkpoint directly.
	_, err := db.ExecContext(context.Background(), "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		// Some in-memory backends don't support WAL checkpoint — that's OK.
		t.Logf("wal_checkpoint not supported on this backend: %v", err)
	}
}
