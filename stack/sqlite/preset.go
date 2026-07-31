package sqlite

import (
	"context"
	"database/sql"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the SQLite preset.
type Option func(*config)

type config struct {
	sqlopt.DSNConfig
	sqlopt.PragmaConfig

	durability     stack.DurabilityTier
	extraStackOpts []stack.Option
}

func defaultConfig() config {
	return config{
		PragmaConfig: sqlopt.PragmaConfig{
			WAL:         true,
			Optimize:    false,
			ForeignKeys: false,
		},
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
		durability:     stack.DurabilityNormal,
		extraStackOpts: nil,
	}
}

// WithDurability sets the durability tier for the SQLite backend. This maps
// to SQLite's PRAGMA synchronous setting:
//
//   - [stack.DurabilityStrict]  → synchronous=FULL (fsync per commit)
//   - [stack.DurabilityNormal]  → synchronous=NORMAL (WAL default — the default)
//   - [stack.DurabilityRelaxed] → synchronous=OFF (no fsync)
//
// The chosen tier is recorded on the Bundle via [stack.WithDurability] so
// benchmark tools can compare backends at the same durability level.
func WithDurability(tier stack.DurabilityTier) Option {
	return func(c *config) { c.durability = tier }
}

// WithPragmas applies shared SQLite PRAGMA options from sqlopt (WAL,
// optimizations, foreign keys):
//
//	b, _ := sqlite.New(dsn, sqlite.WithPragmas(
//	    sqlopt.WithoutWAL(),
//	    sqlopt.WithForeignKeys(),
//	))
func WithPragmas(opts ...sqlopt.PragmaOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.PragmaConfig) }
}

// WithDSN applies shared multi-database DSN options from sqlopt. Use this to
// configure event, query, or view database separation, or to disable
// auto-migration:
//
//	b, _ := sqlite.New(dsn, sqlite.WithDSN(
//	    sqlopt.WithoutAutoMigrate(),
//	    sqlopt.WithEventDB("events.db"),
//	    sqlopt.WithViewDB("views.db"),
//	))
func WithDSN(opts ...sqlopt.DSNOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.DSNConfig) }
}

// WithStack passes additional stack.Options through to the Bundle. Use this to
// add capabilities not covered by the preset itself, such as
// stack.WithMetaEngine:
//
//	meStore, _ := metaengine.Plan(engines, queries)
//	b, _ := sqlite.New(dsn, sqlite.WithStack(stack.WithMetaEngine(meStore)))
func WithStack(opts ...stack.Option) Option {
	return func(c *config) { c.extraStackOpts = append(c.extraStackOpts, opts...) }
}

// New opens a SQLite database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the SQLite connection string — typically a file path ("log.db") or
// ":memory:" for an ephemeral in-process database. The database is opened
// with the pure-Go modernc.org/sqlite driver (no CGo required).
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus uses watermill.EventBus (GoChannel-
// backed, in-process) since SQLite has no pub/sub semantics.
//
// On any setup failure the database is closed before the error is returned —
// no resource leaks. The returned Bundle owns the *sql.DB; Close releases it.
func New(dsn string, opts ...Option) (*stack.Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return newBundle(dsn, cfg)
}

func newBundle(dsn string, cfg config) (*stack.Bundle, error) {
	stackOpts, backend, sqlDB, _, err := sqlopt.InitStack(
		dsn,
		"sqlite",
		cfg.EventDSN,
		cfg.QueryDSN,
		func(d string) (*sql.DB, *storage.SQLBackend, error) { return openBackend(d, cfg) },
		func(d string) (*storage.SQLBackend, io.Closer, error) { return openSecondaryBackend(d, cfg) },
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.init_stack",
			"initialize SQLite stack")
	}

	// Bus is in-process GoChannel (SQLite has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	// Record the durability tier on the Bundle for introspection.
	stackOpts = append(stackOpts, stack.WithDurability(cfg.durability))

	// Extra consumer-provided stack.Options (e.g. stack.WithMetaEngine).
	stackOpts = append(stackOpts, cfg.extraStackOpts...)

	bundle, err := sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "sqlite", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewSQLiteBackend)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.finalize_bundle",
			"finalize SQLite bundle")
	}

	return bundle, nil
}

// openBackend opens the database, applies pragmas and schema, and returns
// both the *sql.DB (for lifecycle) and the SQLBackend (for store access).
// sqliteBusyTimeoutMs is the default busy_timeout for SQLite connections.
// Set via DSN parameter so every pooled connection inherits it (PRAGMAs set
// via db.Exec only apply to the connection that runs them).
const sqliteBusyTimeoutMs = 5000

func openBackend(
	dsn string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	return sqlopt.OpenPrimaryBackend( //nolint:wrapcheck // OpenPrimaryBackend wraps all errors
		func() (*sql.DB, error) {
			return sqlopt.OpenDBOrErr("sqlite",
				storage.EnsureSQLiteDSNBusyTimeout(dsn, sqliteBusyTimeoutMs),
				"sqlite_preset.open_primary")
		},
		func(ctx context.Context, sqlDB *sql.DB) error {
			if cfg.WAL {
				if err := storage.SQLiteEnableWAL(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(err, "sqlite_preset.enable_wal",
						"enable WAL mode")
				}

				// Apply durability tier after WAL setup so the override
				// takes precedence over the NORMAL default.
				if err := sqlopt.ApplySQLiteDurability(ctx, sqlDB, cfg.durability); err != nil {
					return errorfamily.WrapInfrastructure(err, "sqlite_preset.apply_durability",
						"apply durability tier")
				}
			}

			// SQLite WAL serializes writes; capping at 1 connection prevents
			// SQLITE_BUSY errors under concurrent access (see storage.ConfigureSQLitePool).
			storage.ConfigureSQLitePool(sqlDB)

			if cfg.ForeignKeys {
				if err := storage.SQLiteEnableForeignKeys(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(
						err,
						"sqlite_preset.enable_foreign_keys",
						"enable foreign keys",
					)
				}
			}

			if cfg.AutoMigrate {
				if err := storage.SQLiteInitSchema(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(err, "sqlite_preset.init_schema",
						"initialize sqlite schema")
				}
			}

			if cfg.Optimize {
				if err := storage.SQLiteApplyOptimizations(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(
						err,
						"sqlite_preset.apply_optimizations",
						"apply sqlite optimizations",
					)
				}
			}

			return nil
		},
		storage.NewSQLiteBackend,
		"sqlite_preset.create_backend",
	)
}
