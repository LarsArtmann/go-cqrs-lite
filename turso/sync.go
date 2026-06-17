package turso

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tursoclient "turso.tech/database/tursogo"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// syncEngine abstracts the Turso sync operations for testability.
// *tursoclient.TursoSyncDb satisfies this interface in production.
type syncEngine interface {
	Push(ctx context.Context) error
	Pull(ctx context.Context) (bool, error)
	Checkpoint(ctx context.Context) error
	Stats(ctx context.Context) (tursoclient.TursoSyncDbStats, error)
}

// SyncDB wraps a Turso database with remote sync capabilities.
// It provides the *sql.DB for queries and exposes Push/Pull/Checkpoint/Stats
// for sync control.
type SyncDB struct {
	*sql.DB

	syncDb *tursoclient.TursoSyncDb
	engine syncEngine
}

// OpenSync opens a Turso database that syncs with a remote server.
// Local writes work offline. Call Push to send changes, Pull to receive.
//
// For advanced configuration (client name, long-poll timeout, busy timeout,
// namespace, bootstrap behavior), use [OpenSyncWithConfig].
//
// The caller is responsible for closing the returned SyncDB.
func OpenSync(
	ctx context.Context,
	dbPath DbPath,
	remoteURL RemoteURL,
	authToken AuthToken,
) (*SyncDB, error) {
	return OpenSyncWithConfig(ctx, dbPath, remoteURL, authToken)
}

// SyncOption configures advanced sync behavior for [OpenSyncWithConfig].
type SyncOption func(*tursoclient.TursoSyncDbConfig)

// WithSyncNamespace sets the remote namespace for the sync client.
// Namespaces isolate sync state between different applications sharing the
// same Turso database. Empty by default.
func WithSyncNamespace(namespace string) SyncOption {
	return func(c *tursoclient.TursoSyncDbConfig) { c.Namespace = namespace }
}

// WithSyncClientName sets a unique client identifier.
// The sync library uses "turso-sync-go" when omitted. Set this to distinguish
// your application in Turso's sync diagnostics.
func WithSyncClientName(name string) SyncOption {
	return func(c *tursoclient.TursoSyncDbConfig) { c.ClientName = name }
}

// WithSyncLongPollTimeout sets the long-polling timeout used when waiting
// for remote changes. Longer values reduce pull latency at the cost of
// holding connections open. Zero uses the library default.
func WithSyncLongPollTimeout(d time.Duration) SyncOption {
	return func(c *tursoclient.TursoSyncDbConfig) {
		if d > 0 {
			c.LongPollTimeoutMs = int(d.Milliseconds())
		}
	}
}

// WithSyncBootstrapIfEmpty controls whether the initial bootstrap phase
// fetches the full remote state when the local database is empty.
//
// Default is true. Set to false to skip bootstrap and call [SyncDB.Pull]
// explicitly to populate the initial state — useful when you want to control
// when the (potentially large) initial download happens.
func WithSyncBootstrapIfEmpty(bootstrap bool) SyncOption {
	return func(c *tursoclient.TursoSyncDbConfig) { c.BootstrapIfEmpty = &bootstrap }
}

// WithSyncBusyTimeout sets the busy timeout in milliseconds for database
// connections. Default is 5000ms (5 seconds). Set to -1 to disable the busy
// handler.
//
// Under heavy concurrent write load, increase this to give writers more time
// to acquire the write lock before failing with "database is locked".
func WithSyncBusyTimeout(d time.Duration) SyncOption {
	return func(c *tursoclient.TursoSyncDbConfig) {
		c.BusyTimeout = int(d.Milliseconds())
	}
}

// OpenSyncWithConfig opens a Turso sync database with advanced configuration.
// It is the configurable variant of [OpenSync].
//
// In-memory databases cannot sync and will return [ErrMemorySync].
//
// The caller is responsible for closing the returned SyncDB.
func OpenSyncWithConfig(
	ctx context.Context,
	dbPath DbPath,
	remoteURL RemoteURL,
	authToken AuthToken,
	opts ...SyncOption,
) (*SyncDB, error) {
	if remoteURL != "" && strings.HasPrefix(string(dbPath), ":memory:") {
		return nil, event.WrapRejection(
			ErrMemorySync,
			"storage.turso_memory_sync",
			fmt.Sprintf(
				"in-memory databases lose data on restart when using remote sync: got %q",
				dbPath,
			),
		)
	}

	cfg := tursoclient.TursoSyncDbConfig{ //nolint:exhaustruct // only required fields; others use library defaults
		Path:      string(dbPath),
		RemoteUrl: string(remoteURL),
		AuthToken: string(authToken),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	syncDb, err := tursoclient.NewTursoSyncDb(ctx, cfg)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.create_turso_sync",
			"create turso sync db for "+string(remoteURL))
	}

	database, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.connect_turso_sync",
			"connect turso sync db for "+string(remoteURL))
	}

	return &SyncDB{DB: database, syncDb: syncDb, engine: syncDb}, nil
}

// Push sends local writes to the remote Turso server.
func (t *SyncDB) Push(ctx context.Context) error {
	err := t.engine.Push(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_push",
			"turso push")
	}

	return nil
}

// Pull fetches remote changes into the local database.
// Returns true if any changes were received.
func (t *SyncDB) Pull(ctx context.Context) (bool, error) {
	changed, err := t.engine.Pull(ctx)
	if err != nil {
		return changed, event.WrapInfrastructure(err, "storage.turso_pull",
			"turso pull")
	}

	return changed, nil
}

// Checkpoint writes the WAL into the main database file.
func (t *SyncDB) Checkpoint(ctx context.Context) error {
	err := t.engine.Checkpoint(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_checkpoint",
			"turso checkpoint")
	}

	return nil
}

// Close closes the underlying SQL database connection.
// Does not disconnect from remote sync — Push/Pull will fail after Close.
func (t *SyncDB) Close() error {
	return t.DB.Close() //nolint:wrapcheck // sql.DB.Close is self-descriptive
}

// Stats returns sync statistics (WAL size, bytes sent/received).
func (t *SyncDB) Stats(ctx context.Context) (tursoclient.TursoSyncDbStats, error) {
	stats, err := t.engine.Stats(ctx)
	if err != nil {
		return stats, event.WrapInfrastructure(err, "storage.turso_stats",
			"turso stats")
	}

	return stats, nil
}

// HealthCheck verifies that the underlying database connection is usable.
// It runs a lightweight SELECT 1 via [database/sql.DB.PingContext].
//
// This does NOT verify remote sync connectivity — use [SyncDB.Pull] or
// [SyncDB.Stats] to confirm the remote server is reachable. HealthCheck is
// suitable for liveness/readiness probes in orchestrators (k8s, Nomad, etc.).
//
// Returns nil if the database responds, an [event.Infrastructure] error otherwise.
func (t *SyncDB) HealthCheck(ctx context.Context) error {
	err := t.PingContext(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_health_check",
			"ping turso database")
	}

	return nil
}

// SyncClient returns the underlying Turso sync client, granting access to
// low-level operations not exposed by SyncDB (e.g., direct access to
// experimental features, custom polling loops). Most consumers do not need
// this — the SyncDB methods cover the common cases.
func (t *SyncDB) SyncClient() *tursoclient.TursoSyncDb {
	return t.syncDb
}

// newSyncDBWithEngine creates a SyncDB with a custom sync engine for testing.
// The engine replaces the real Turso sync client so Push/Pull/Checkpoint/Stats
// can be exercised without a live server. syncDb is nil — SyncClient() returns nil.
func newSyncDBWithEngine(db *sql.DB, engine syncEngine) *SyncDB {
	return &SyncDB{
		DB:     db,
		syncDb: nil,
		engine: engine,
	}
}

// Backward-compatible aliases.
//
//nolint:gochecknoglobals // backward-compatible aliases
var (
	OpenTursoSync = OpenSync
)

// TursoSyncDB is a backward-compatible alias for SyncDB.
type TursoSyncDB = SyncDB
