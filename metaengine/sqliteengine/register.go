package sqliteengine

import (
	"context"

	_ "modernc.org/sqlite" // SQLite driver registered with database/sql

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver(
		"sqlite",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			// Owning variant: the driver factory cannot hand the *sql.DB to
			// the caller, so the engine must close it on Close.
			// Engine API takes no ctx (same as benchkit/phases_metaengine_sqlite.go).
			return NewSQLiteEngineFromDSN(cfg.DSN, cfg.Pragmas...) //nolint:contextcheck,wrapcheck
		},
	)
}
