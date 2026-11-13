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
			// Materialized-view specs flow through so unsupported engines fail
			// construction loudly instead of silently ignoring the operator.
			// MaxOpenConns flows through only when raised (>0): the
			// serialized default is deliberate, so zero stays zero here and
			// NewSQLiteEngineFromDSNWith keeps MaxOpenConns(1).
			opts := []EngineOption{
				WithMaterializedViews(cfg.MaterializedViews),
				WithMaxOpenConns(cfg.MaxOpenConns),
			}

			// ADR-0141 temporal cells: a non-nil policy switches the engine
			// into versioned-cell mode with the operator's retention bound
			// (zero policy = keep everything).
			if cfg.CellVersioning != nil {
				if err := cfg.CellVersioning.Validate(); err != nil {
					return nil, fmt.Errorf("driver %q cell_versioning: %w", "sqlite", err)
				}

				opts = append(opts, WithCellVersioning(WithRetention(*cfg.CellVersioning)))
			}

			return NewSQLiteEngineFromDSNWith(cfg.DSN, pragmas,
				opts...) //nolint:contextcheck,wrapcheck
		},
	)
}
