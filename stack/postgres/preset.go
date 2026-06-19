package postgres

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

// Option configures the Postgres preset.
type Option func(*config)

type config struct {
	autoMigrate bool
}

func defaultConfig() config {
	return config{autoMigrate: true}
}

// WithoutAutoMigrate skips schema creation. Use this when you manage schemas
// yourself (e.g. via a migration tool). By default New creates all required tables.
func WithoutAutoMigrate() Option {
	return func(c *config) { c.autoMigrate = false }
}

// New opens a PostgreSQL database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the PostgreSQL connection string, e.g.
// "postgres://user:pass@localhost:5432/myapp?sslmode=disable". The database is
// opened with the pure-Go pgx driver (no CGo required).
//
// Events, commands, queries, snapshots, and checkpoints are persisted to the
// database. The event bus uses an in-memory implementation (memory.NewMemoryBus)
// since PostgreSQL has no pub/sub semantics. Read models use an in-memory KV
// store (kv.MemStore).
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
	db, err := sql.Open("pgx", dsn) //nolint:varnamelen
	if err != nil {
		return nil, fmt.Errorf("postgres preset: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.autoMigrate {
		err = storage.PostgresInitSchema(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("postgres preset: init schema: %w", err)
		}
	}

	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("postgres preset: create backend: %w", err)
	}

	stackOpts := buildOptions(backend)

	stackOpts = append(stackOpts, stack.WithBus(memory.NewMemoryBus()))
	stackOpts = append(stackOpts, stack.WithReadModels(kv.NewMemStore()))
	stackOpts = append(
		stackOpts,
		stack.WithCloser(backend),
		stack.WithCloser(&funcCloser{fn: db.Close}),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()
		_ = db.Close()

		return nil, fmt.Errorf("postgres preset: wire bundle: %w", err)
	}

	return b, nil
}

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

type funcCloser struct {
	fn func() error
}

func (c *funcCloser) Close() error { return c.fn() }

var _ io.Closer = (*funcCloser)(nil)
