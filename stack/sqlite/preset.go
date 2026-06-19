package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// Option configures the SQLite preset.
type Option func(*config)

type config struct {
	wal         bool
	autoMigrate bool
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

// New opens a SQLite database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the SQLite connection string — typically a file path ("log.db") or
// ":memory:" for an ephemeral in-process database. The database is opened
// with the pure-Go modernc.org/sqlite driver (no CGo required).
//
// Events, commands, queries, snapshots, and checkpoints are persisted to the
// database. The event bus uses an in-memory implementation (memory.NewMemoryBus)
// since SQLite has no pub/sub semantics. Read models use an in-memory KV store
// (kv.MemStore); a SQL-backed KV adapter is future work.
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

	// Read models are in-memory until a SQL-backed kv.Store exists.
	stackOpts = append(stackOpts, stack.WithReadModels(kv.NewMemStore()))

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

		return nil, err
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

	if cmdStore, err := backend.CommandStore(); err == nil {
		opts = append(opts, stack.WithCommandStore(cmdStore))
	}

	if queryStore, err := backend.QueryStore(); err == nil {
		opts = append(opts, stack.WithQueryStore(queryStore))
	}

	if snapStore, err := backend.SnapshotStore(); err == nil {
		opts = append(opts, stack.WithSnapshotStore(snapStore))
	}

	if cpStore, err := backend.CheckpointStore(); err == nil {
		opts = append(opts, stack.WithCheckpointStore(cpStore))
	}

	return opts
}

// openBackend opens the database, applies pragmas and schema, and returns
// both the *sql.DB (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (*sql.DB, *storage.SQLBackend, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.wal {
		if err := storage.SQLiteEnableWAL(ctx, db); err != nil {
			_ = db.Close()
			return nil, nil, fmt.Errorf("sqlite: enable WAL: %w", err)
		}
	}

	if cfg.autoMigrate {
		if err := storage.SQLiteInitSchema(ctx, db); err != nil {
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
