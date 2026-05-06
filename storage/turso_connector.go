package storage

import (
	"context"
	"database/sql"
	"fmt"

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
	syncDb, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
		Path:       dbPath,
		RemoteUrl:  remoteURL,
		AuthToken:  authToken,
	})
	if err != nil {
		return nil, fmt.Errorf("create turso sync db: %w", err)
	}

	db, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect turso sync db: %w", err)
	}

	return &TursoSyncDB{DB: db, syncDb: syncDb}, nil
}

// Push sends local writes to the remote Turso server.
func (t *TursoSyncDB) Push(ctx context.Context) error {
	return t.syncDb.Push(ctx)
}

// Pull fetches remote changes into the local database.
// Returns true if any changes were received.
func (t *TursoSyncDB) Pull(ctx context.Context) (bool, error) {
	return t.syncDb.Pull(ctx)
}

// Checkpoint writes the WAL into the main database file.
func (t *TursoSyncDB) Checkpoint(ctx context.Context) error {
	return t.syncDb.Checkpoint(ctx)
}

// Stats returns sync statistics (WAL size, bytes sent/received).
func (t *TursoSyncDB) Stats(ctx context.Context) (turso.TursoSyncDbStats, error) {
	return t.syncDb.Stats(ctx)
}
