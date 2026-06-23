package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// Option configures the Postgres preset.
type Option func(*config)

type config struct {
	autoMigrate bool
	eventDSN    string                       // override DSN for event store
	queryDSN    string                       // override DSN for query/command audit store
	viewDSN     string                       // override DSN for read-model KV store
	listener    storage.NotificationListener // nil → in-memory bus
	busOpts     []storage.PostgresBusOption  // forwarded when listener != nil
}

func defaultConfig() config {
	return config{ //nolint:exhaustruct // options fill fields
		autoMigrate: true,
		eventDSN:    "",
		queryDSN:    "",
		viewDSN:     "",
	}
}

// WithoutAutoMigrate skips schema creation. Use this when you manage schemas
// yourself (e.g. via a migration tool). By default New creates all required tables.
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

// WithDistributedBus enables cross-process event propagation via Postgres
// LISTEN/NOTIFY. The listener is typically a [PgxListener] constructed from
// the same database DSN. When this option is set, the preset wires
// [storage.PostgresBus] instead of the default in-memory bus; the listener
// is registered with the Bundle for Close-time cleanup.
//
// Optional busOpts (e.g. [storage.WithBusChannel], [storage.WithRefetchAttempts])
// are forwarded to [storage.NewPostgresBus] when the distributed bus is active.
// They are ignored when no listener is set.
//
// Without this option, the preset uses watermill.EventBus (GoChannel) — fine
// for single-process deployments but invisible to other processes sharing the DB.
func WithDistributedBus(
	listener storage.NotificationListener,
	busOpts ...storage.PostgresBusOption,
) Option {
	return func(c *config) {
		c.listener = listener
		c.busOpts = busOpts
	}
}

// New opens a PostgreSQL database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the PostgreSQL connection string, e.g.
// "postgres://user:pass@localhost:5432/myapp?sslmode=disable". The database is
// opened with the pure-Go pgx driver (no CGo required).
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. By default the event bus is watermill.EventBus
// (GoChannel, in-process) for single-process use; pass [WithDistributedBus]
// to wire storage.PostgresBus (LISTEN/NOTIFY) for multi-process pub/sub.
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

	// Override: event-sourcing stores (events, snapshots, checkpoints) from a
	// separate database if configured.
	if cfg.eventDSN != "" {
		eventOpts, eventCloser, eErr := openEventStores(cfg.eventDSN, cfg)
		if eErr != nil {
			_ = backend.Close()
			_ = db.Close()

			return nil, fmt.Errorf("postgres preset: open event db: %w", eErr)
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
			_ = db.Close()

			return nil, fmt.Errorf("postgres preset: open query db: %w", qErr)
		}

		stackOpts = append(stackOpts, queryOpts...)
		stackOpts = append(stackOpts, stack.WithCloser(queryCloser))
	}

	bus, busCleanup, err := buildBus(db, backend.EventStore(), cfg)
	if err != nil {
		_ = backend.Close()
		_ = db.Close()

		return nil, fmt.Errorf("postgres preset: bus: %w", err)
	}

	stackOpts = append(stackOpts, stack.WithBus(bus))

	if busCleanup != nil {
		stackOpts = append(stackOpts, stack.WithCloser(busCleanup))
	}

	viewOpts, err := buildViewOptions(cfg, backend, db)
	if err != nil {
		return nil, err
	}

	stackOpts = append(stackOpts, viewOpts...)

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

// openBackend opens the database, applies schema, and returns both the *sql.DB
// (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (*sql.DB, *storage.SQLBackend, error) {
	db, err := sql.Open("pgx", dsn) //nolint:varnamelen
	if err != nil {
		return nil, nil, fmt.Errorf("postgres preset: open %q: %w", dsn, err)
	}

	ctx := context.Background()

	if cfg.autoMigrate {
		err = storage.PostgresInitSchema(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, nil, fmt.Errorf("postgres preset: init schema: %w", err)
		}
	}

	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		_ = db.Close()

		return nil, nil, fmt.Errorf("postgres preset: create backend: %w", err)
	}

	return db, backend, nil
}

// buildBus returns the event bus to wire into the Bundle. When cfg.listener
// is set, the bus is a storage.PostgresBus backed by Postgres LISTEN/NOTIFY
// and the listener is returned so the caller can register it for Close-time
// cleanup. Otherwise the bus is watermill.EventBus (GoChannel) for
// single-process use.
func buildBus(
	dbHandle *sql.DB,
	store event.EventSource,
	cfg config,
) (event.Bus, io.Closer, error) {
	if cfg.listener == nil {
		return cqrswatermill.NewEventBus(), nil, nil
	}

	pgBus, err := storage.NewPostgresBus(dbHandle, store, cfg.listener, cfg.busOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create postgres bus: %w", err)
	}

	return pgBus, pgBus, nil
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
