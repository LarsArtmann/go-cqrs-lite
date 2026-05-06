package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLiteSnapshotStore persists aggregate snapshots in a SQLite database.
type SQLiteSnapshotStore struct {
	db *sql.DB
}

// NewSQLiteSnapshotStore creates a new SQLite-backed snapshot store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteSnapshotStore(db *sql.DB) (*SQLiteSnapshotStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLiteSnapshotStore{db: db}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLiteSnapshotStore) Close() error { return nil }

// SQLiteSnapshotSchema returns the SQL DDL for creating the snapshots table in SQLite.
func SQLiteSnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

// Save persists a snapshot for an aggregate.
func (s *SQLiteSnapshotStore) Save(ctx context.Context, snap event.Snapshot) error {
	query := `INSERT INTO snapshots (aggregate_type, aggregate_id, version, state, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (aggregate_type, aggregate_id)
		DO UPDATE SET version = excluded.version, state = excluded.state, created_at = excluded.created_at`

	_, err := s.db.ExecContext(
		ctx,
		query,
		string(snap.AggregateType),
		snap.AggregateID,
		snap.Version.Int(),
		snap.State,
		snap.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("save snapshot for %s %s: %w", snap.AggregateType, snap.AggregateID, err)
	}

	return nil
}

// Load retrieves the latest snapshot for an aggregate.
// Returns ErrSnapshotNotFound if no snapshot exists.
func (s *SQLiteSnapshotStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	query := `SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = ? AND aggregate_id = ?`

	snap, err := sqliteScanSnapshot(
		s.db.QueryRowContext(ctx, query, string(aggregateType), aggregateID),
		aggregateType,
		aggregateID,
	)
	if err != nil {
		return nil, fmt.Errorf("load snapshot for %s %s: %w", aggregateType, aggregateID, err)
	}

	return snap, nil
}

// LoadAtVersion retrieves a snapshot at or before a specific version.
func (s *SQLiteSnapshotStore) LoadAtVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	query := `SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = ? AND aggregate_id = ?`

	snap, err := sqliteScanSnapshot(
		s.db.QueryRowContext(ctx, query, string(aggregateType), aggregateID),
		aggregateType,
		aggregateID,
	)
	if err != nil {
		return nil, fmt.Errorf("load snapshot at version %d for %s %s: %w",
			version, aggregateType, aggregateID, err)
	}

	if snap.Version.Int() > version.Int() {
		return nil, fmt.Errorf("load snapshot at version %d for %s %s: %w",
			version, aggregateType, aggregateID, event.ErrSnapshotNotFound)
	}

	return snap, nil
}

// Delete removes a snapshot for an aggregate.
func (s *SQLiteSnapshotStore) Delete(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	query := `DELETE FROM snapshots WHERE aggregate_type = ? AND aggregate_id = ?`

	_, err := s.db.ExecContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return fmt.Errorf("delete snapshot for %s %s: %w", aggregateType, aggregateID, err)
	}

	return nil
}

var _ event.SnapshotStore = (*SQLiteSnapshotStore)(nil)
