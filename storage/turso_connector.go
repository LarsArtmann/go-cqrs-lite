package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	turso "turso.tech/database/tursogo"
)

// OpenTurso opens a local Turso database file and returns a *sql.DB
// compatible with all SQLite* adapters in this package.
//
// The caller is responsible for closing the returned *sql.DB.
func OpenTurso(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("turso", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open turso database at %s: %w", dbPath, err)
	}

	return db, nil
}

// OpenTursoInMemory opens an in-memory Turso database and returns a *sql.DB.
// Useful for testing and development.
func OpenTursoInMemory() (*sql.DB, error) {
	return OpenTurso(":memory:")
}

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
		return nil, fmt.Errorf(
			"%w: got %q: in-memory databases lose data on restart when using remote sync",
			ErrTursoMemorySync,
			dbPath,
		)
	}

	syncDb, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
		Path:      dbPath,
		RemoteUrl: remoteURL,
		AuthToken: authToken,
	})
	if err != nil {
		return nil, fmt.Errorf("create turso sync db for %s: %w", remoteURL, err)
	}

	db, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect turso sync db for %s: %w", remoteURL, err)
	}

	return &TursoSyncDB{DB: db, syncDb: syncDb}, nil
}

// Push sends local writes to the remote Turso server.
func (t *TursoSyncDB) Push(ctx context.Context) error {
	err := t.syncDb.Push(ctx)
	if err != nil {
		return fmt.Errorf("turso push: %w", err)
	}

	return nil
}

// Pull fetches remote changes into the local database.
// Returns true if any changes were received.
func (t *TursoSyncDB) Pull(ctx context.Context) (bool, error) {
	changed, err := t.syncDb.Pull(ctx)
	if err != nil {
		return changed, fmt.Errorf("turso pull: %w", err)
	}

	return changed, nil
}

// Checkpoint writes the WAL into the main database file.
func (t *TursoSyncDB) Checkpoint(ctx context.Context) error {
	err := t.syncDb.Checkpoint(ctx)
	if err != nil {
		return fmt.Errorf("turso checkpoint: %w", err)
	}

	return nil
}

// Stats returns sync statistics (WAL size, bytes sent/received).
func (t *TursoSyncDB) Stats(ctx context.Context) (turso.TursoSyncDbStats, error) {
	stats, err := t.syncDb.Stats(ctx)
	if err != nil {
		return stats, fmt.Errorf("turso stats: %w", err)
	}

	return stats, nil
}
