package turso

import (
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// openSecondaryBackend opens and configures a secondary Turso database,
// creates its backend, and returns both along with a closer that releases
// the backend and the *sql.DB. Shared by the event-DB and query-DB paths.
func openSecondaryBackend(
	dbPath string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	return sqlopt.NewSecondaryBackend(dbPath,
		func() (*sql.DB, error) { return openSecondaryDB(dbPath, cfg) },
		cqrsturso.NewBackend,
		"turso.create_secondary_backend")
}

// openSecondaryDB opens and configures a secondary Turso database.
func openSecondaryDB(dbPath string, cfg config) (*sql.DB, error) {
	sqlDB, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "turso.open_secondary",
			fmt.Sprintf("open %q", dbPath))
	}

	cqrsturso.ConfigurePool(sqlDB)

	if err := applySchemaAndPragmas(sqlDB, cfg); err != nil {
		_ = sqlDB.Close()

		return nil, err
	}

	return sqlDB, nil
}
