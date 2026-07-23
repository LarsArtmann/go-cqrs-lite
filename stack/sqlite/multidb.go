package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// openSecondaryDB opens and configures a secondary SQLite database (for events,
// queries, or views when multi-DB mode is enabled via WithEventDB etc.).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "sqlite.open",
			fmt.Sprintf("open %q", dsn))
	}

	ctx := context.Background()

	if cfg.WAL {
		err = storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "sqlite.enable_wal",
				fmt.Sprintf("enable WAL on %q", dsn))
		}
	}

	if cfg.ForeignKeys {
		err = storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "sqlite.enable_fk",
				fmt.Sprintf("enable foreign keys on %q", dsn))
		}
	}

	if cfg.AutoMigrate {
		err = storage.SQLiteInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "sqlite.init_schema",
				fmt.Sprintf("init schema on %q", dsn))
		}
	}

	if cfg.Optimize {
		err = storage.SQLiteApplyOptimizations(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "sqlite.optimize",
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
	backend, closer, err := sqlopt.NewSecondaryBackend(dsn,
		func() (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewSQLiteBackend,
		"sqlite.create_backend")
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite.create_secondary_backend",
			"create secondary SQLite backend")
	}

	return backend, closer, nil
}
