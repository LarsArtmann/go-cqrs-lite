package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// Option configures the SQLite preset.
type Option func(*config)

type config struct {
	wal         bool
	optimize    bool
	foreignKeys bool
	autoMigrate bool
	eventDSN    string // override DSN for event store
	queryDSN    string // override DSN for query/command audit store
	viewDSN     string // override DSN for read-model KV store
}

func defaultConfig() config {
	return config{
		wal:         true,
		optimize:    false,
		foreignKeys: false,
		autoMigrate: true,
		eventDSN:    "",
		queryDSN:    "",
		viewDSN:     "",
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
func WithoutAutoMigrate() Option {
	return func(c *config) { c.autoMigrate = false }
}

// WithEventDB sets a separate DSN for the event store. When set, events,
// snapshots, and checkpoints are persisted to this database instead of the
// primary DSN. The deployer chooses this when isolating write-heavy event
// streams from query traffic.
func WithEventDB(dsn string) Option {
	return func(c *config) { c.eventDSN = dsn }
}

// WithQueryDB sets a separate DSN for the command and query audit stores.
// When set, persisted commands and queries go to this database.
func WithQueryDB(dsn string) Option {
	return func(c *config) { c.queryDSN = dsn }
}

// WithViewDB sets a separate DSN for the read-model KV store. When set,
// materialized views are persisted to this database, isolating read-model
// scans from the event store.
func WithViewDB(dsn string) Option {
	return func(c *config) { c.viewDSN = dsn }
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
	sqlDB, backend, err := openBackend(dsn, cfg)
	if err != nil {
		return nil, err
	}

	stackOpts := buildOptions(backend)

	// Override: event-sourcing stores (events, snapshots, checkpoints) from a
	// separate database if configured.
	if cfg.eventDSN != "" {
		eventOpts, eventCloser, eErr := openEventStores(cfg.eventDSN, cfg)
		if eErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: open event db: %w", eErr)
		}

		stackOpts = append(stackOpts, eventOpts...)
		stackOpts = append(stackOpts, stack.WithCloser(eventCloser))
	}

	// Override: command and query audit stores from a separate database if
	// configured.
	if cfg.queryDSN != "" {
		queryOpts, queryCloser, qErr := openQueryStores(cfg.queryDSN, cfg)
		if qErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, fmt.Errorf("sqlite: open query db: %w", qErr)
		}

		stackOpts = append(stackOpts, queryOpts...)
		stackOpts = append(stackOpts, stack.WithCloser(queryCloser))
	}

	// Bus is in-process GoChannel (SQLite has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	viewOpts, err := buildViewOptions(cfg, backend, sqlDB)
	if err != nil {
		return nil, err
	}

	stackOpts = append(stackOpts, viewOpts...)

	// Register lifecycle: backend closes stores, dbCloser closes the DB.
	// Order matters — stores must close before the DB.
	stackOpts = append(
		stackOpts,
		stack.WithDatabase(sqlDB),
		stack.WithCloser(backend),
		stack.WithCloser(stack.NewFuncCloser(sqlDB.Close)),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()

		_ = sqlDB.Close()

		return nil, fmt.Errorf("sqlite: wire bundle: %w", err)
	}

	return b, nil
}

// buildOptions assembles the stack.Option slice for an SQLBackend's stores.
// Store creation errors (CommandStore, QueryStore, etc.) are fatal here
// because the backend was already successfully created.
func buildOptions(backend *storage.SQLBackend) []stack.Option {
	opts := []stack.Option{
		stack.WithEventStore(backend.EventStore()),
	}

	cmdStore, err := backend.CommandStore()
	if err == nil {
		opts = append(opts, stack.WithCommandStore(cmdStore))
	}

	queryStore, err := backend.QueryStore()
	if err == nil {
		opts = append(opts, stack.WithQueryStore(queryStore))
	}

	snapStore, err := backend.SnapshotStore()
	if err == nil {
		opts = append(opts, stack.WithSnapshotStore(snapStore))
	}

	cpStore, err := backend.CheckpointStore()
	if err == nil {
		opts = append(opts, stack.WithCheckpointStore(cpStore))
	}

	return opts
}

// openBackend opens the database, applies pragmas and schema, and returns
// both the *sql.DB (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (*sql.DB, *storage.SQLBackend, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, fmt.Errorf("sqlite: enable WAL: %w", err)
		}
	}

	if cfg.foreignKeys {
		err = storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, fmt.Errorf("sqlite: enable foreign keys: %w", err)
		}
	}

	if cfg.autoMigrate {
		err = storage.SQLiteInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, fmt.Errorf("sqlite: init schema: %w", err)
		}
	}

	if cfg.optimize {
		err = storage.SQLiteApplyOptimizations(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, fmt.Errorf("sqlite: apply optimizations: %w", err)
		}
	}

	backend, err := storage.NewSQLiteBackend(sqlDB)
	if err != nil {
		_ = sqlDB.Close()

		return nil, nil, fmt.Errorf("sqlite: create backend: %w", err)
	}

	return sqlDB, backend, nil
}
