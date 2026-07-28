package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// openSecondaryDB opens and configures a secondary DuckDB database (for events,
// queries, or views when multi-DB mode is enabled via WithEventDB etc.).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	dsn = appendDuckDBOptions(dsn, cfg)

	sqlDB, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "duckdb.open",
			fmt.Sprintf("open %q", dsn))
	}

	if cfg.AutoMigrate {
		ctx := context.Background()

		if err := storage.DuckDBInitSchema(ctx, sqlDB); err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "duckdb.init_schema",
				fmt.Sprintf("init schema on %q", dsn))
		}
	}

	return sqlDB, nil
}

// openSecondaryBackend opens and configures a secondary DuckDB database,
// creates its backend, and returns both along with a closer that releases
// the backend and the *sql.DB. Shared by the event-DB and query-DB paths.
func openSecondaryBackend(
	dsn string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	backend, closer, err := sqlopt.NewSecondaryBackend(dsn,
		func() (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewDuckDBBackend,
		"duckdb.create_backend")
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "duckdb.create_secondary_backend",
			"create secondary DuckDB backend")
	}

	return backend, closer, nil
}
