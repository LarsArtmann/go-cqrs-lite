package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLCheckpointStore persists projection checkpoints in a SQL database.
type SQLCheckpointStore struct {
	db *sql.DB
}

// NewSQLCheckpointStore creates a new SQL-backed checkpoint store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewSQLCheckpointStore(db *sql.DB) *SQLCheckpointStore {
	return &SQLCheckpointStore{db: db}
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLCheckpointStore) Close() error { return nil }

// CheckpointSchema returns the SQL DDL for creating the checkpoints table.
func CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR(255) PRIMARY KEY,
    event_id        TEXT NOT NULL
);`
}

// Load returns the last processed event ID for a projection.
func (s *SQLCheckpointStore) Load(ctx context.Context, projectionName string) (id.EventID, error) {
	query := `SELECT event_id FROM checkpoints WHERE projection_name = $1`

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
func (s *SQLCheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	eventID id.EventID,
) error {
	query := `INSERT INTO checkpoints (projection_name, event_id)
		VALUES ($1, $2)
		ON CONFLICT (projection_name)
		DO UPDATE SET event_id = EXCLUDED.event_id`

	_, err := s.db.ExecContext(ctx, query, projectionName, eventID)
	if err != nil {
		return fmt.Errorf("save checkpoint for projection %q: %w", projectionName, err)
	}

	return nil
}

var _ event.CheckpointStore = (*SQLCheckpointStore)(nil)
