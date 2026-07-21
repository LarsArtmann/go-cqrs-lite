package sqlite

import (
	"context"
	"database/sql"
	"fmt"
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

	wal         bool
	optimize    bool
	foreignKeys bool
}

func defaultConfig() config {
	return config{
		wal:         true,
		optimize:    false,
		foreignKeys: false,
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
		},
	}
}

// WithoutWAL disables WAL mode. By default New enables WAL plus a busy
// timeout of 5 seconds to eliminate "database is locked" errors under
// concurrency.
func WithoutWAL() Option {
	return func(c *config) { c.wal = false }
}

// WithOptimizations applies performance PRAGMAs tuned for CQRS workloads:
// cache_size (64 MB), temp_store=MEMORY, and mmap_size (256 MB). These
// improve throughput without durability trade-offs. Recommended for
// production. Has no effect if [WithoutAutoMigrate] is also set.
func WithOptimizations() Option {
	return func(c *config) { c.optimize = true }
}

// WithForeignKeys enables SQLite foreign-key enforcement on all databases
// (primary and secondary when multi-DB is used). Off by default because
// existing databases may contain orphaned references that would cause errors
// once enforcement is active. Enable for new databases where referential
// integrity is required.
func WithForeignKeys() Option {
	return func(c *config) { c.foreignKeys = true }
}

// WithoutAutoMigrate skips schema creation. Use this when you manage schemas
// yourself (e.g. via a migration tool). By default New creates all required
// tables.
//
// The 4-option block (WithoutAutoMigrate, WithEventDB, WithQueryDB, WithViewDB)
// is duplicated in stack/postgres/preset.go. This is forced by Go's typed
// Option pattern: each preset has `type Option func(*config)` where config is
// preset-specific (sqlite adds wal/optimize/foreignKeys; postgres adds
// listener/busOpts). The bodies are 1-line delegates to the shared
// sqlopt.DSNConfig methods, but the wrapper function shape must be per-preset
// because Go function types are invariant (func(*sqlite.config) ≠
// func(*postgres.config)).
func WithoutAutoMigrate() Option {
	return func(c *config) { c.WithoutAutoMigrate() }
}

// WithEventDB sets a separate DSN for the event store. When set, events,
// snapshots, and checkpoints are persisted to this database instead of the
// primary DSN. The deployer chooses this when isolating write-heavy event
// streams from query traffic.
func WithEventDB(dsn string) Option {
	return func(c *config) { c.SetEventDB(dsn) }
}

// WithQueryDB sets a separate DSN for the command and query audit stores.
// When set, persisted commands and queries go to this database.
func WithQueryDB(dsn string) Option {
	return func(c *config) { c.SetQueryDB(dsn) }
}

// WithViewDB sets a separate DSN for the read-model KV store. When set,
// materialized views are persisted to this database, isolating read-model
// scans from the event store.
func WithViewDB(dsn string) Option {
	return func(c *config) { c.SetViewDB(dsn) }
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
		return nil, err
	}

	// Bus is in-process GoChannel (SQLite has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	return sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "sqlite", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewSQLiteBackend)
}

// openBackend opens the database, applies pragmas and schema, and returns
// both the *sql.DB (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (*sql.DB, *storage.SQLBackend, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.open_primary",
			fmt.Sprintf("open sqlite %q", dsn))
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.enable_wal",
				"enable WAL mode")
		}
	}

	if cfg.foreignKeys {
		err = storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, errorfamily.WrapInfrastructure(
				err,
				"sqlite_preset.enable_foreign_keys",
				"enable foreign keys",
			)
		}
	}

	if cfg.AutoMigrate {
		err = storage.SQLiteInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.init_schema",
				"initialize sqlite schema")
		}
	}

	if cfg.optimize {
		err = storage.SQLiteApplyOptimizations(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, errorfamily.WrapInfrastructure(
				err,
				"sqlite_preset.apply_optimizations",
				"apply sqlite optimizations",
			)
		}
	}

	backend, err := storage.NewSQLiteBackend(sqlDB)
	if err != nil {
		_ = sqlDB.Close()

		return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.create_backend",
			"create SQL backend")
	}

	return sqlDB, backend, nil
}
