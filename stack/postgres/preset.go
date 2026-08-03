package postgres

import (
	"context"
	"database/sql"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the Postgres preset.
type Option func(*config)

type config struct {
	sqlopt.DSNConfig

	durability       stack.DurabilityTier
	maxOpenConns     int
	maxIdleConns     int
	statementTimeout time.Duration
}

func defaultConfig() config {
	return config{
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
		durability:       stack.DurabilityNormal,
		maxOpenConns:     0,
		maxIdleConns:     0,
		statementTimeout: 0,
	}
}

// WithDurability sets the durability tier for the Postgres backend. This maps
// to Postgres's synchronous_commit setting:
//
//   - [stack.DurabilityStrict]  → synchronous_commit=on (fsync WAL per commit)
//   - [stack.DurabilityNormal]  → synchronous_commit=off (no per-commit fsync,
//     ~200x faster writes, small window of lost transactions on OS crash)
//   - [stack.DurabilityRelaxed] → synchronous_commit=off (same as Normal for
//     Postgres)
//
// The setting is injected into the connection DSN so every pooled connection
// inherits it — not just the connection that first opens the database.
//
// The chosen tier is recorded on the Bundle via [stack.WithDurability] so
// benchmark tools can compare backends at the same durability level.
func WithDurability(tier stack.DurabilityTier) Option {
	return func(c *config) { c.durability = tier }
}

// WithPoolSize sets the maximum number of open and idle connections in the
// Postgres connection pool. By default, database/sql uses unlimited open
// connections and 2 idle connections. For write-heavy CQRS workloads, a
// smaller pool (e.g. 10-20) reduces contention and memory overhead.
//
// Set maxOpen to 0 for unlimited (the database/sql default).
// Set maxIdle to -1 for no idle limit.
func WithPoolSize(maxOpen, maxIdle int) Option {
	return func(c *config) {
		c.maxOpenConns = maxOpen
		c.maxIdleConns = maxIdle
	}
}

// WithStatementTimeout sets the maximum duration a query can run before
// Postgres aborts it. Injected into the connection DSN (statement_timeout=<ms>)
// so every pooled connection inherits it. Zero or negative disables the
// timeout (the Postgres default).
func WithStatementTimeout(d time.Duration) Option {
	return func(c *config) { c.statementTimeout = d }
}

// WithDSN applies shared multi-database DSN options from sqlopt. Use this to
// configure event, query, or view database separation, or to disable
// auto-migration:
//
//	b, _ := postgres.New(dsn, postgres.WithDSN(
//	    sqlopt.WithoutAutoMigrate(),
//	    sqlopt.WithEventDB("postgres://host/events_db"),
//	    sqlopt.WithViewDB("postgres://host/views_db"),
//	))
func WithDSN(opts ...sqlopt.DSNOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.DSNConfig) }
}

// New opens a PostgreSQL database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the PostgreSQL connection string, e.g.
// "postgres://user:pass@localhost:5432/myapp?sslmode=disable". The database is
// opened with the pure-Go pgx driver (no CGo required).
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus is watermill.EventBus (GoChannel,
// in-process) for single-process use.
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
	stackOpts, backend, sqlDB, closePrimary, err := sqlopt.InitStack(
		dsn,
		"postgres",
		cfg.EventDSN,
		cfg.QueryDSN,
		func(d string) (*sql.DB, *storage.SQLBackend, error) { return openBackend(d, cfg) },
		func(d string) (*storage.SQLBackend, io.Closer, error) { return openSecondaryBackend(d, cfg) },
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.init_stack",
			"initialize Postgres stack")
	}

	bus := cqrswatermill.NewEventBus()

	stackOpts = append(stackOpts, stack.WithBus(bus))
	stackOpts = append(stackOpts, stack.WithDurability(cfg.durability))
	stackOpts = append(stackOpts, stack.WithCapabilities(stack.Capabilities{
		Backend:     "postgres",
		Persistent:  true,
		Distributed: false,
		DurabilityRange: []stack.DurabilityTier{
			stack.DurabilityStrict,
			stack.DurabilityNormal,
			stack.DurabilityRelaxed,
		},
		OLAP:        false,
		CGoRequired: false,
		Embedded:    false,
		SyncEnabled: false,
	}))

	bundle, err := sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "postgres", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewSQLBackend)
	if err != nil {
		closePrimary()

		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.finalize_bundle",
			"finalize Postgres bundle")
	}

	return bundle, nil
}

// applyDSNSettings injects durability and statement_timeout into the DSN so
// every pooled connection inherits them. Session-level SET commands only apply
// to one connection — with MaxOpenConns > 1, new connections would silently
// revert to server defaults. pgx applies DSN-level GUCs on every new
// connection, making this pool-safe.
func applyDSNSettings(dsn string, cfg config) string {
	if cfg.durability != "" {
		dsn = storage.EnsurePostgresSynchronousCommit(
			dsn, cfg.durability == stack.DurabilityStrict,
		)
	}

	if cfg.statementTimeout > 0 {
		dsn = storage.EnsurePostgresStatementTimeout(
			dsn, cfg.statementTimeout.Milliseconds(),
		)
	}

	return dsn
}

// openBackend opens the database, applies schema, and returns both the *sql.DB
// (for lifecycle) and the SQLBackend (for store access).
func openBackend(
	dsn string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	return sqlopt.OpenPrimaryBackend( //nolint:wrapcheck // OpenPrimaryBackend wraps all errors
		func() (*sql.DB, error) {
			return sqlopt.OpenDBOrErr(
				"pgx",
				applyDSNSettings(dsn, cfg),
				"postgres_preset.open_primary",
			)
		},
		func(ctx context.Context, sqlDB *sql.DB) error {
			// Apply pool sizing before any connections are created.
			if cfg.maxOpenConns > 0 {
				sqlDB.SetMaxOpenConns(cfg.maxOpenConns)
			}

			if cfg.maxIdleConns >= 0 && cfg.maxIdleConns != 0 {
				sqlDB.SetMaxIdleConns(cfg.maxIdleConns)
			}

			// Durability and statement_timeout are injected at the DSN level
			// (see applyDSNSettings) so every pooled connection inherits them.
			// Session-level SET only applies to one connection.

			if !cfg.AutoMigrate {
				return nil
			}

			if err := storage.PostgresInitSchema(ctx, sqlDB); err != nil {
				return errorfamily.WrapInfrastructure(err, "postgres_preset.init_schema",
					"initialize postgres schema")
			}

			return nil
		},
		storage.NewSQLBackend,
		"postgres_preset.create_backend",
	)
}
