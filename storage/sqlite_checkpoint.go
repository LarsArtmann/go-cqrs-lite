package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLiteCheckpointStore persists projection checkpoints in a SQLite database.
type SQLiteCheckpointStore struct {
	db *sql.DB
}

// NewSQLiteCheckpointStore creates a new SQLite-backed checkpoint store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteCheckpointStore(db *sql.DB) (*SQLiteCheckpointStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLiteCheckpointStore{db: db}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLiteCheckpointStore) Close() error { return nil }

// SQLiteCheckpointSchema returns the SQL DDL for creating the checkpoints table in SQLite.
func SQLiteCheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name TEXT PRIMARY KEY,
    event_id        TEXT NOT NULL
);`
}

// Load returns the last processed event ID for a projection.
func (s *SQLiteCheckpointStore) Load(ctx context.Context, projectionName string) (id.EventID, error) {
	query := `SELECT event_id FROM checkpoints WHERE projection_name = ?`

	var eventIDStr string

	err := s.db.QueryRowContext(ctx, query, projectionName).Scan(&eventIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return id.EventID{}, nil
		}

		return id.EventID{}, fmt.Errorf(
			"load checkpoint for projection %q: %w",
			projectionName,
			err,
		)
	}

	parsed, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return id.EventID{}, fmt.Errorf(
			"parse event ID %q for projection %q: %w",
			eventIDStr,
			projectionName,
			err,
		)
	}

	return parsed, nil
}

// Save persists the last processed event ID for a projection.
func (s *SQLiteCheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	eventID id.EventID,
) error {
	query := `INSERT INTO checkpoints (projection_name, event_id)
		VALUES (?, ?)
		ON CONFLICT (projection_name)
		DO UPDATE SET event_id = excluded.event_id`

	_, err := s.db.ExecContext(ctx, query, projectionName, eventID)
	if err != nil {
		return fmt.Errorf("save checkpoint for projection %q: %w", projectionName, err)
	}

	return nil
}

var _ event.CheckpointStore = (*SQLiteCheckpointStore)(nil)
