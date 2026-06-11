package turso

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	tursoclient "turso.tech/database/tursogo"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// SyncDB wraps a Turso database with remote sync capabilities.
// It provides the *sql.DB for queries and exposes Push/Pull/Checkpoint/Stats
// for sync control.
type SyncDB struct {
	*sql.DB

	syncDb *tursoclient.TursoSyncDb
}

// OpenSync opens a Turso database that syncs with a remote server.
// Local writes work offline. Call Push to send changes, Pull to receive.
//
// The caller is responsible for closing the returned SyncDB.
func OpenSync(
	ctx context.Context,
	dbPath DbPath,
	remoteURL RemoteURL,
	authToken AuthToken,
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

	syncDb, err := tursoclient.NewTursoSyncDb(
		ctx,
		tursoclient.TursoSyncDbConfig{ //nolint:exhaustruct // only required fields; others use library defaults
			Path:      string(dbPath),
			RemoteUrl: string(remoteURL),
			AuthToken: string(authToken),
		},
	)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.create_turso_sync",
			"create turso sync db for "+string(remoteURL))
	}

	database, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.connect_turso_sync",
			"connect turso sync db for "+string(remoteURL))
	}

	return &SyncDB{DB: database, syncDb: syncDb}, nil
}

// Push sends local writes to the remote Turso server.
func (t *SyncDB) Push(ctx context.Context) error {
	err := t.syncDb.Push(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_push",
			"turso push")
	}

	return nil
}

// Pull fetches remote changes into the local database.
// Returns true if any changes were received.
func (t *SyncDB) Pull(ctx context.Context) (bool, error) {
	changed, err := t.syncDb.Pull(ctx)
	if err != nil {
		return changed, event.WrapInfrastructure(err, "storage.turso_pull",
			"turso pull")
	}

	return changed, nil
}

// Checkpoint writes the WAL into the main database file.
func (t *SyncDB) Checkpoint(ctx context.Context) error {
	err := t.syncDb.Checkpoint(ctx)
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
	stats, err := t.syncDb.Stats(ctx)
	if err != nil {
		return stats, event.WrapInfrastructure(err, "storage.turso_stats",
			"turso stats")
	}

	return stats, nil
}

// Backward-compatible aliases.
//
//nolint:gochecknoglobals // backward-compatible aliases
var (
	OpenTursoSync = OpenSync
)

// TursoSyncDB is a backward-compatible alias for SyncDB.
type TursoSyncDB = SyncDB
