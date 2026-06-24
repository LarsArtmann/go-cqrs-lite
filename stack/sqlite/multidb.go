package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryDB opens and configures a secondary SQLite database (for events,
// queries, or views when multi-DB mode is enabled via WithEventDB etc.).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: enable WAL on %q: %w", dsn, err)
		}
	}

	if cfg.foreignKeys {
		err = storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: enable foreign keys on %q: %w", dsn, err)
		}
	}

	if cfg.autoMigrate {
		err = storage.SQLiteInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: init schema on %q: %w", dsn, err)
		}
	}

	if cfg.optimize {
		err = storage.SQLiteApplyOptimizations(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: apply optimizations on %q: %w", dsn, err)
		}
	}

	return sqlDB, nil
}

// openSecondaryBackend opens and configures a secondary SQLite database,
// creates its backend, and returns both along with a closer that releases
// the backend and the *sql.DB. Shared by the event-DB and query-DB paths.
func openSecondaryBackend(
	dsn string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openSecondaryDB(dsn, cfg)
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := storage.NewSQLiteBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, fmt.Errorf("sqlite: create backend for %q: %w", dsn, err)
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}
