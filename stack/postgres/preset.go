package postgres

import (
	"context"
	"database/sql"
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
	sqlopt.DSNConfig

	listener storage.NotificationListener // nil → in-memory bus
	busOpts  []storage.PostgresBusOption  // forwarded when listener != nil
}

func defaultConfig() config {
	return config{
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
		listener: nil,
		busOpts:  nil,
	}
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

	bus, busCleanup, err := buildBus(sqlDB, backend.EventStore(), cfg)
	if err != nil {
		closePrimary()

		return nil, errorfamily.WrapInfrastructure(err, "postgres_preset.bus",
			"build event bus")
	}

	stackOpts = append(stackOpts, stack.WithBus(bus))

	if busCleanup != nil {
		stackOpts = append(stackOpts, stack.WithCloser(busCleanup))
	}

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

// openBackend opens the database, applies schema, and returns both the *sql.DB
// (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (db *sql.DB, backend *storage.SQLBackend, err error) {
	db, err = sqlopt.OpenDBOrErr("pgx", dsn, "postgres_preset.open_primary")
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil && db != nil {
			_ = db.Close()
		}
	}()

	ctx := context.Background()

	if cfg.AutoMigrate {
		if err = storage.PostgresInitSchema(ctx, db); err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.init_schema",
				"initialize postgres schema")
		}
	}

	backend, err = storage.NewSQLBackend(db)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.create_backend",
			"create SQL backend")
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
		return nil, nil, errorfamily.WrapInfrastructure(err, "postgres_preset.create_bus",
			"create postgres LISTEN/NOTIFY bus")
	}

	return pgBus, pgBus, nil
}
