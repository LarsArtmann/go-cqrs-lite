package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryDB opens and configures a secondary SQLite database (for events,
// queries, or views when multi-DB mode is enabled via WithEventDB etc.).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "sqlite.open",
			fmt.Sprintf("open %q", dsn))
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, event.WrapInfrastructure(err, "sqlite.enable_wal",
				fmt.Sprintf("enable WAL on %q", dsn))
		}
	}

	if cfg.foreignKeys {
		err = storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, event.WrapInfrastructure(err, "sqlite.enable_fk",
				fmt.Sprintf("enable foreign keys on %q", dsn))
		}
	}

	if cfg.autoMigrate {
		err = storage.SQLiteInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, event.WrapInfrastructure(err, "sqlite.init_schema",
				fmt.Sprintf("init schema on %q", dsn))
		}
	}

	if cfg.optimize {
		err = storage.SQLiteApplyOptimizations(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, event.WrapInfrastructure(err, "sqlite.optimize",
				fmt.Sprintf("apply optimizations on %q", dsn))
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

		return nil, nil, event.WrapInfrastructure(err, "sqlite.create_backend",
			fmt.Sprintf("create backend for %q", dsn))
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}
