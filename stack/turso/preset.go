package turso

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// Option configures the Turso preset.
type Option func(*config)

type config struct {
	autoMigrate bool
	eventPath   string // override path for event store
	queryPath   string // override path for query/command audit store
	viewPath    string // override path for read-model KV store
	syncOpts    []cqrsturso.SyncOption
}

func defaultConfig() config {
	return config{autoMigrate: true}
}

// WithoutAutoMigrate skips schema creation. Use this when you manage schemas
// yourself (e.g. via a migration tool). By default New creates all required
// tables.
func WithoutAutoMigrate() Option {
	return func(c *config) { c.autoMigrate = false }
}

// WithEventDB sets a separate database path for the event store. When set,
// events, snapshots, and checkpoints are persisted to this database instead of
// the primary path. The deployer chooses this when isolating write-heavy event
// streams from query traffic.
func WithEventDB(path string) Option {
	return func(c *config) { c.eventPath = path }
}

// WithQueryDB sets a separate database path for the command and query audit
// stores. When set, persisted commands and queries go to this database.
func WithQueryDB(path string) Option {
	return func(c *config) { c.queryPath = path }
}

// WithViewDB sets a separate database path for the read-model KV store. When
// set, materialized views are persisted to this database, isolating read-model
// scans from the event store.
func WithViewDB(path string) Option {
	return func(c *config) { c.viewPath = path }
}

// WithSyncOptions passes advanced configuration to the underlying Turso sync
// client. Use this with [NewSync] to set client name, long-poll timeout, busy
// timeout, namespace, or bootstrap behavior. Has no effect with [New].
func WithSyncOptions(opts ...cqrsturso.SyncOption) Option {
	return func(c *config) { c.syncOpts = opts }
}

// Bundle wraps [stack.Bundle] with Turso-specific sync capabilities. It embeds
// *stack.Bundle, so all Bundle fields and methods are available directly.
type Bundle struct {
	*stack.Bundle

	syncDB *cqrsturso.SyncDB // nil for local-only deployments
}

// Sync returns the Turso sync database for remote sync operations (Push, Pull,
// Checkpoint, Stats, HealthCheck). Returns nil for local-only deployments
// created with [New].
func (b *Bundle) Sync() *cqrsturso.SyncDB {
	return b.syncDB
}

// New opens a local Turso (embedded LibSQL) database at dbPath and returns a
// fully-wired [Bundle].
//
// dbPath is a filesystem path for the LibSQL database file (e.g. "app.db").
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus uses an in-process GoChannel since
// LibSQL has no pub/sub semantics.
//
// On any setup failure the database is closed before the error is returned —
// no resource leaks. The returned Bundle owns the database; Close releases it.
func New(dbPath string, opts ...Option) (*Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return newLocalBundle(dbPath, cfg)
}

// NewSync opens a Turso database with remote sync and returns a fully-wired
// [Bundle]. The local database works offline; call [Bundle.Sync] to access
// Push/Pull/Checkpoint/Stats for synchronization with the remote server.
//
// dbPath is a filesystem path for the local LibSQL file. remoteURL is the
// Turso server URL (e.g. "libsql://my-db.turso.io"). authToken is the Turso
// auth token.
//
// Multi-database options ([WithEventDB], [WithQueryDB], [WithViewDB]) are not
// supported in sync mode — the entire CQRS stack shares one syncing database.
// Use [WithSyncOptions] for advanced sync configuration.
func NewSync(
	ctx context.Context,
	dbPath, remoteURL, authToken string,
	opts ...Option,
) (*Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return newSyncBundle(ctx, dbPath, remoteURL, authToken, cfg)
}

func newLocalBundle(dbPath string, cfg config) (*Bundle, error) {
	sqlDB, backend, err := openLocalBackend(dbPath, cfg)
	if err != nil {
		return nil, err
	}

	stackOpts := buildOptions(backend)

	// Override: event-sourcing stores (events, snapshots, checkpoints) from a
	// separate database if configured.
	if cfg.eventPath != "" {
		eventOpts, eventCloser, eErr := openEventStores(cfg.eventPath, cfg)
		if eErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, fmt.Errorf("turso: open event db: %w", eErr)
		}

		stackOpts = append(stackOpts, eventOpts...)
		stackOpts = append(stackOpts, stack.WithCloser(eventCloser))
	}

	// Override: command and query audit stores from a separate database if
	// configured.
	if cfg.queryPath != "" {
		queryOpts, queryCloser, qErr := openQueryStores(cfg.queryPath, cfg)
		if qErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, fmt.Errorf("turso: open query db: %w", qErr)
		}

		stackOpts = append(stackOpts, queryOpts...)
		stackOpts = append(stackOpts, stack.WithCloser(queryCloser))
	}

	// Bus is in-process GoChannel (LibSQL has no pub/sub).
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
		stack.WithCloser(backend),
		stack.WithCloser(&funcCloser{fn: sqlDB.Close}),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()

		_ = sqlDB.Close()

		return nil, fmt.Errorf("turso: wire bundle: %w", err)
	}

	return &Bundle{Bundle: b}, nil
}

func newSyncBundle(
	ctx context.Context,
	dbPath, remoteURL, authToken string,
	cfg config,
) (*Bundle, error) {
	syncDB, err := cqrsturso.OpenSyncWithConfig(
		ctx,
		cqrsturso.DbPath(dbPath),
		cqrsturso.RemoteURL(remoteURL),
		cqrsturso.AuthToken(authToken),
		cfg.syncOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("turso: open sync db: %w", err)
	}

	sqlDB := syncDB.DB

	cqrsturso.ConfigurePool(sqlDB)

	ctxBg := context.Background()

	if cfg.autoMigrate {
		err := cqrsturso.InitSchema(ctxBg, sqlDB)
		if err != nil {
			_ = syncDB.Close()

			return nil, fmt.Errorf("turso: init schema: %w", err)
		}
	}

	backend, err := cqrsturso.NewBackend(sqlDB)
	if err != nil {
		_ = syncDB.Close()

		return nil, fmt.Errorf("turso: create backend: %w", err)
	}

	stackOpts := buildOptions(backend)
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	viewOpts, err := buildPrimaryViewOptions(backend, sqlDB)
	if err != nil {
		return nil, err
	}

	stackOpts = append(stackOpts, viewOpts...)

	// Register lifecycle: backend closes stores, syncDB closes the DB +
	// disconnects sync. Order matters — stores must close before the DB.
	stackOpts = append(
		stackOpts,
		stack.WithCloser(backend),
		stack.WithCloser(syncDB),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()
		_ = syncDB.Close()

		return nil, fmt.Errorf("turso: wire bundle: %w", err)
	}

	return &Bundle{Bundle: b, syncDB: syncDB}, nil
}

// openLocalBackend opens the database, configures the pool, applies the schema,
// and returns both the *sql.DB (for lifecycle) and the Backend (for stores).
func openLocalBackend(
	dbPath string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	sqlDB, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		return nil, nil, fmt.Errorf("turso: open %q: %w", dbPath, err)
	}

	cqrsturso.ConfigurePool(sqlDB)

	if cfg.autoMigrate {
		ctx := context.Background()

		err := cqrsturso.InitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, nil, fmt.Errorf("turso: init schema: %w", err)
		}
	}

	backend, err := cqrsturso.NewBackend(sqlDB)
	if err != nil {
		_ = sqlDB.Close()

		return nil, nil, fmt.Errorf("turso: create backend: %w", err)
	}

	return sqlDB, backend, nil
}

// buildOptions assembles the stack.Option slice for a Backend's stores.
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
