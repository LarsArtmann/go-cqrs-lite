package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// openSecondaryDB opens and configures a secondary Postgres database (for
// events, queries, or views when multi-DB mode is enabled).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "postgres.open_secondary",
			fmt.Sprintf("open %q", dsn))
	}

	if cfg.AutoMigrate {
		ctx := context.Background()

		err = storage.PostgresInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err, "postgres.init_schema",
				fmt.Sprintf("init schema on %q", dsn))
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

		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres.create_backend",
			fmt.Sprintf("create backend for %q", dsn))
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}
