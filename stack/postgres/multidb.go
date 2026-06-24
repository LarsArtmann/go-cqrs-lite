package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryDB opens and configures a secondary Postgres database (for
// events, queries, or views when multi-DB mode is enabled).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres preset: open %q: %w", dsn, err)
	}

	if cfg.autoMigrate {
		ctx := context.Background()

		err = storage.PostgresInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("postgres preset: init schema on %q: %w", dsn, err)
		}
	}

	return sqlDB, nil
}

// openSecondaryBackend opens a secondary Postgres database, creates its
// backend, and returns both along with a closer. Shared by the event-DB and
// query-DB paths.
func openSecondaryBackend(
	dsn string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openSecondaryDB(dsn, cfg)
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := storage.NewSQLBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, fmt.Errorf("postgres preset: create backend for %q: %w", dsn, err)
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}
