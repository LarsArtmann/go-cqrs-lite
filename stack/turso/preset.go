package turso

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the Turso preset.
type Option func(*config)

type config struct {
	autoMigrate bool
	optimize    bool
	foreignKeys bool
	wal         bool
	eventPath   string // override path for event store
	queryPath   string // override path for query/command audit store
	viewPath    string // override path for read-model KV store
	syncOpts    []cqrsturso.SyncOption
}

func defaultConfig() config {
	return config{autoMigrate: true, optimize: false, foreignKeys: false, wal: true}
}

// WithoutWAL disables WAL mode. By default New enables WAL plus a busy
// timeout of 5 seconds to eliminate "database is locked" errors under
// concurrency. WAL is safe for single-writer workloads and improves read
// concurrency significantly.
func WithoutWAL() Option {
	return func(c *config) { c.wal = false }
}

// WithoutAutoMigrate skips schema creation. Use this when you manage schemas
// yourself (e.g. via a migration tool). By default New creates all required
// tables.
func WithoutAutoMigrate() Option {
	return func(c *config) { c.autoMigrate = false }
}

// WithOptimizations applies CQRS-optimized indexes and performance PRAGMAs
// after schema creation. This calls
// [turso.InitSchemaWithIndexesAndOptimizations] instead of plain
// [turso.InitSchema], adding recommended indexes on event/store tables and
// setting PRAGMAs tuned for CQRS workloads. Recommended for production.
//
// Has no effect if [WithoutAutoMigrate] is also set.
func WithOptimizations() Option {
	return func(c *config) { c.optimize = true }
}

// WithForeignKeys enables foreign-key enforcement on all databases (primary
// and secondary when multi-DB is used). Off by default because existing
// databases may contain orphaned references. Enable for new databases where
// referential integrity is required.
func WithForeignKeys() Option {
	return func(c *config) { c.foreignKeys = true }
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

// New opens a local Turso (embedded database) at dbPath and returns a
// fully-wired [Bundle].
//
// dbPath is a filesystem path for the Turso database file (e.g. "app.db").
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus uses an in-process GoChannel since
// the embedded database has no pub/sub semantics.
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
// dbPath is a filesystem path for the local Turso database file. remoteURL is the
// Turso server URL (e.g. "https://my-db.turso.io"). authToken is the Turso
// auth token.
//
// Multi-database options ([WithEventDB], [WithQueryDB], [WithViewDB]) are
// incompatible with sync mode — the entire CQRS stack must share one syncing
// database so that Push/Pull replicates all data consistently. Passing any
// of them returns an error.
//
// Use [WithSyncOptions] for advanced sync configuration (client name, long-poll
// timeout, busy timeout, namespace, bootstrap behavior).
func NewSync(
	ctx context.Context,
	dbPath, remoteURL, authToken string,
	opts ...Option,
) (*Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.eventPath != "" || cfg.queryPath != "" || cfg.viewPath != "" {
		return nil, errorfamily.NewRejection("turso_preset.multi_db_incompatible",
			"turso: multi-DB options (WithEventDB, WithQueryDB, WithViewDB) "+
				"are incompatible with NewSync — all stores must share one "+
				"syncing database")
	}

	return newSyncBundle(ctx, dbPath, remoteURL, authToken, cfg)
}

func newLocalBundle(dbPath string, cfg config) (*Bundle, error) {
	sqlDB, backend, err := openLocalBackend(dbPath, cfg)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.open_local_backend",
			"open local backend")
	}

	stackOpts := sqlopt.AllOptions(backend)
	stackOpts = append(stackOpts, stack.WithDatabase(sqlDB))

	// Override: event-sourcing stores (events, snapshots, checkpoints) from a
	// separate database if configured.
	if cfg.eventPath != "" {
		evtBackend, evtCloser, eErr := openSecondaryBackend(cfg.eventPath, cfg)
		if eErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(eErr, "turso_preset.open_event_db",
				"open event database")
		}

		stackOpts = append(stackOpts, sqlopt.EventStoreOptions(evtBackend)...)
		stackOpts = append(stackOpts, stack.WithCloser(evtCloser))
	}

	// Override: command and query audit stores from a separate database if
	// configured.
	if cfg.queryPath != "" {
		qBackend, qCloser, qErr := openSecondaryBackend(cfg.queryPath, cfg)
		if qErr != nil {
			_ = backend.Close()
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(qErr, "turso_preset.open_query_db",
				"open query database")
		}

		stackOpts = append(stackOpts, sqlopt.QueryStoreOptions(qBackend)...)
		stackOpts = append(stackOpts, stack.WithCloser(qCloser))
	}

	// Bus is in-process GoChannel (the embedded database has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	viewOpts, err := buildViewOptions(cfg, backend, sqlDB)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.view_options",
			"build view options")
	}

	stackOpts = append(stackOpts, viewOpts...)

	// Register lifecycle: backend closes stores, dbCloser closes the DB.
	// Order matters — stores must close before the DB.
	stackOpts = append(
		stackOpts,
		stack.WithCloser(backend),
		stack.WithCloser(stack.NewFuncCloser(sqlDB.Close)),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()

		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.wire_local_bundle",
			"wire local turso bundle")
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
		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.open_sync_db",
			"open syncing turso database")
	}

	sqlDB := syncDB.DB

	cqrsturso.ConfigurePool(sqlDB)

	if err := applySchemaAndPragmas(sqlDB, cfg); err != nil {
		_ = syncDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.schema_pragmas",
			"apply schema and pragmas")
	}

	backend, err := cqrsturso.NewBackend(sqlDB)
	if err != nil {
		_ = syncDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.create_backend",
			"create turso backend")
	}

	stackOpts := sqlopt.AllOptions(backend)
	stackOpts = append(stackOpts, stack.WithDatabase(sqlDB))
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	kvStore, err := backend.KVStore()
	if err != nil {
		_ = backend.Close()
		_ = syncDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.kv_store",
			"create KV store")
	}

	stackOpts = append(stackOpts, stack.WithReadModels(kvStore))

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

		return nil, errorfamily.WrapInfrastructure(err, "turso_preset.wire_sync_bundle",
			"wire sync turso bundle")
	}

	return &Bundle{Bundle: b, syncDB: syncDB}, nil
}
