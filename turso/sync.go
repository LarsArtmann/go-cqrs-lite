package turso

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	turso "turso.tech/database/tursogo"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// TursoSyncDB wraps a Turso database with remote sync capabilities.
// It provides the *sql.DB for queries and exposes Push/Pull/Checkpoint/Stats
// for sync control.
type TursoSyncDB struct {
	*sql.DB

	syncDb *turso.TursoSyncDb
}

// OpenTursoSync opens a Turso database that syncs with a remote server.
// Local writes work offline. Call Push to send changes, Pull to receive.
//
// The caller is responsible for closing the returned TursoSyncDB.
func OpenTursoSync(ctx context.Context, dbPath, remoteURL, authToken string) (*TursoSyncDB, error) {
	if remoteURL != "" && strings.HasPrefix(dbPath, ":memory:") {
		return nil, event.WrapRejection(
			ErrTursoMemorySync,
			"storage.turso_memory_sync",
			fmt.Sprintf(
				"in-memory databases lose data on restart when using remote sync: got %q",
				dbPath,
			),
		)
	}

	syncDb, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
		Path:      dbPath,
		RemoteUrl: remoteURL,
		AuthToken: authToken,
	})
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.create_turso_sync",
			"create turso sync db for "+remoteURL)
	}

	db, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.connect_turso_sync",
			"connect turso sync db for "+remoteURL)
	}

	return &TursoSyncDB{DB: db, syncDb: syncDb}, nil
}

// Push sends local writes to the remote Turso server.
func (t *TursoSyncDB) Push(ctx context.Context) error {
	err := t.syncDb.Push(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_push",
			"turso push")
	}

	return nil
}

// Pull fetches remote changes into the local database.
// Returns true if any changes were received.
func (t *TursoSyncDB) Pull(ctx context.Context) (bool, error) {
	changed, err := t.syncDb.Pull(ctx)
	if err != nil {
		return changed, event.WrapInfrastructure(err, "storage.turso_pull",
			"turso pull")
	}

	return changed, nil
}

// Checkpoint writes the WAL into the main database file.
func (t *TursoSyncDB) Checkpoint(ctx context.Context) error {
	err := t.syncDb.Checkpoint(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.turso_checkpoint",
			"turso checkpoint")
	}

	return nil
}

// Close closes the underlying SQL database connection.
// Does not disconnect from remote sync — Push/Pull will fail after Close.
func (t *TursoSyncDB) Close() error {
	return t.DB.Close()
}

// Stats returns sync statistics (WAL size, bytes sent/received).
func (t *TursoSyncDB) Stats(ctx context.Context) (turso.TursoSyncDbStats, error) {
	stats, err := t.syncDb.Stats(ctx)
	if err != nil {
		return stats, event.WrapInfrastructure(err, "storage.turso_stats",
			"turso stats")
	}

	return stats, nil
}
