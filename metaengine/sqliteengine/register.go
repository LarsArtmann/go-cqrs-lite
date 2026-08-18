package sqliteengine

import (
	"context"

	_ "modernc.org/sqlite" // SQLite driver registered with database/sql

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"sqlite",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			pragmas, err := durabilityPragmas(cfg.Durability, cfg.Pragmas)
			if err != nil {
				return nil, err
			}

			// Owning variant: the driver factory cannot hand the *sql.DB to
			// the caller, so the engine must close it on Close.
			// Engine API takes no ctx (same as benchkit/phases_metaengine_sqlite.go).
			return NewSQLiteEngineFromDSN(cfg.DSN, pragmas...) //nolint:contextcheck,wrapcheck
		},
	)
}
