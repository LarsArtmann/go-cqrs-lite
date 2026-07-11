package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
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
	sqlDB, backend, err := openBackend(dsn, cfg)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.open_backend",
			"open primary backend")
	}

	stackOpts := sqlopt.AllOptions(backend)

	// Override: event-sourcing stores (events, snapshots, checkpoints) from a
	// separate database if configured.
	if cfg.eventDSN != "" {
		evtBackend, evtCloser, eErr := openSecondaryBackend(cfg.eventDSN, cfg)
		if eErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(eErr, "postgres_preset.open_event_db",
				"open event database")
		}

		stackOpts = append(stackOpts, sqlopt.EventStoreOptions(evtBackend)...)
		stackOpts = append(stackOpts, stack.WithCloser(evtCloser))
	}

	// Override: command and query audit stores from a separate database if
	// configured.
	if cfg.queryDSN != "" {
		qBackend, qCloser, qErr := openSecondaryBackend(cfg.queryDSN, cfg)
		if qErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(qErr, "postgres_preset.open_query_db",
				"open query database")
		}

		stackOpts = append(stackOpts, sqlopt.QueryStoreOptions(qBackend)...)
		stackOpts = append(stackOpts, stack.WithCloser(qCloser))
	}

	bus, busCleanup, err := buildBus(sqlDB, backend.EventStore(), cfg)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.bus",
			"build event bus")
	}

	stackOpts = append(stackOpts, stack.WithBus(bus))

	if busCleanup != nil {
		stackOpts = append(stackOpts, stack.WithCloser(busCleanup))
	}

	viewOpts, err := buildViewOptions(cfg, backend, sqlDB)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.view_options",
			"build view options")
	}

	stackOpts = append(stackOpts, viewOpts...)

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

		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.wire_bundle",
			"wire postgres bundle")
	}

	return b, nil
}

// openBackend opens the database, applies schema, and returns both the *sql.DB
// (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (*sql.DB, *storage.SQLBackend, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.open_primary",
			fmt.Sprintf("open postgres %q", dsn))
	}

	ctx := context.Background()

	if cfg.autoMigrate {
		err = storage.PostgresInitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.init_schema",
				"initialize postgres schema")
		}
	}

	backend, err := storage.NewSQLBackend(sqlDB)
	if err != nil {
		_ = sqlDB.Close()

		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.create_backend",
			"create SQL backend")
	}

	return sqlDB, backend, nil
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
		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.create_bus",
			"create postgres LISTEN/NOTIFY bus")
	}

	return pgBus, pgBus, nil
}
