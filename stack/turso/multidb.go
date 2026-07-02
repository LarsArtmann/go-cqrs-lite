package turso

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryBackend opens and configures a secondary Turso database,
// creates its backend, and returns both along with a closer that releases
// the backend and the *sql.DB. Shared by the event-DB and query-DB paths.
func openSecondaryBackend(
	dbPath string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openSecondaryDB(dbPath, cfg)
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := cqrsturso.NewBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, event.WrapInfrastructure(err, "turso.create_secondary_backend",
			fmt.Sprintf("create backend for %q", dbPath))
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}

// openSecondaryDB opens and configures a secondary Turso database.
func openSecondaryDB(dbPath string, cfg config) (*sql.DB, error) {
	sqlDB, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		return nil, event.WrapInfrastructure(err, "turso.open_secondary",
			fmt.Sprintf("open %q", dbPath))
	}

	cqrsturso.ConfigurePool(sqlDB)

	if err := applySchemaAndPragmas(sqlDB, cfg); err != nil {
		_ = sqlDB.Close()

		return nil, err
	}

	return sqlDB, nil
}
