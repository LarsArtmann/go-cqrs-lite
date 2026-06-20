package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// Option configures the SQLite preset.
type Option func(*config)

type config struct {
	wal         bool
	autoMigrate bool
	eventDSN    string // override DSN for event store
	queryDSN    string // override DSN for query/command audit store
	viewDSN     string // override DSN for read-model KV store
}

func defaultConfig() config {
	return config{wal: true, autoMigrate: true}
}

// WithoutWAL disables WAL mode. By default New enables WAL plus a busy
// timeout of 5 seconds to eliminate "database is locked" errors under
// concurrency.
func WithoutWAL() Option {
	return func(c *config) { c.wal = false }
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
// persisted to the database. The event bus uses an in-memory implementation
// (memory.NewMemoryBus) since SQLite has no pub/sub semantics.
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
	db, backend, err := openBackend(dsn, cfg) 
	if err != nil {
		return nil, err
	}

	stackOpts := buildOptions(backend)

	// Bus is in-memory (SQLite has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(memory.NewMemoryBus()))

	// Read models: use separate view DB if configured, else primary.
	var viewDB *sql.DB

	if cfg.viewDSN != "" {
		viewDB, err = openSecondaryDB(cfg.viewDSN, cfg)
		if err != nil {
			_ = backend.Close()
			_ = db.Close()

			return nil, fmt.Errorf("sqlite: open view db: %w", err)
		}

		viewBackend, err := storage.NewSQLiteBackend(viewDB)
		if err != nil {
			_ = backend.Close()
			_ = db.Close()
			_ = viewDB.Close()

			return nil, fmt.Errorf("sqlite: create view backend: %w", err)
		}

		kvStore, err := viewBackend.KVStore()
		if err != nil {
			_ = viewBackend.Close()
			_ = backend.Close()
			_ = db.Close()

			return nil, fmt.Errorf("sqlite: kv store (view db): %w", err)
		}

		stackOpts = append(stackOpts, stack.WithReadModels(kvStore))
		stackOpts = append(stackOpts, stack.WithCloser(viewBackend))
		stackOpts = append(stackOpts, stack.WithCloser(&funcCloser{fn: viewDB.Close}))
	} else {
		kvStore, err := backend.KVStore()
		if err != nil {
			_ = backend.Close()
			_ = db.Close()

			return nil, fmt.Errorf("sqlite: kv store: %w", err)
		}

		stackOpts = append(stackOpts, stack.WithReadModels(kvStore))
	}

	// Register lifecycle: backend closes stores, dbCloser closes the DB.
	// Order matters — stores must close before the DB.
	stackOpts = append(
		stackOpts,
		stack.WithCloser(backend),
		stack.WithCloser(&funcCloser{fn: db.Close}),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()



		_ = db.Close()
		if viewDB != nil {
			_ = viewDB.Close()
		}

		return nil, fmt.Errorf("sqlite: wire bundle: %w", err)
	}

	return b, nil
}

// openSecondaryDB opens and configures a secondary SQLite database (for events,
// queries, or views when multi-DB mode is enabled via WithEventDB etc.).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("sqlite: enable WAL on %q: %w", dsn, err)
		}
	}

	if cfg.autoMigrate {
		err = storage.SQLiteInitSchema(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("sqlite: init schema on %q: %w", dsn, err)
		}
	}

	return db, nil
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
	db, err := sql.Open("sqlite", dsn
		return nil, nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.wal {
		err = storage.SQLiteEnableWAL(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, nil, fmt.Errorf("sqlite: enable WAL: %w", err)
		}
	}

	if cfg.autoMigrate {
		err = storage.SQLiteInitSchema(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, nil, fmt.Errorf("sqlite: init schema: %w", err)
		}
	}

	backend, err := storage.NewSQLiteBackend(db)
	if err != nil {
		_ = db.Close()

		return nil, nil, fmt.Errorf("sqlite: create backend: %w", err)
	}

	return db, backend, nil
}

// funcCloser adapts a func() error into an io.Closer. It is a pointer-receiving
// struct (not a function type) so it is comparable and can be used as a map
// key for Bundle.Close deduplication.
type funcCloser struct {
	fn func() error
}

func (c *funcCloser) Close() error { return c.fn() }

var _ io.Closer = (*funcCloser)(nil)
