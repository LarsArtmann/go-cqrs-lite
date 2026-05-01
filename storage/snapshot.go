package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLSnapshotStore persists aggregate snapshots in a SQL database.
type SQLSnapshotStore struct {
	db *sql.DB
}

// NewSQLSnapshotStore creates a new SQL-backed snapshot store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewSQLSnapshotStore(db *sql.DB) *SQLSnapshotStore {
	return &SQLSnapshotStore{db: db}
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLSnapshotStore) Close() error { return nil }

// SnapshotSchema returns the SQL DDL for creating the snapshots table.
func SnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           JSONB NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

// Save persists a snapshot for an aggregate.
// State is stored as-is ([]byte) — no additional marshaling is applied.
func (s *SQLSnapshotStore) Save(ctx context.Context, snap event.Snapshot) error {
	query := `INSERT INTO snapshots (aggregate_type, aggregate_id, version, state, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (aggregate_type, aggregate_id)
		DO UPDATE SET version = EXCLUDED.version, state = EXCLUDED.state, created_at = EXCLUDED.created_at`

	_, err := s.db.ExecContext(
		ctx,
		query,
		string(snap.AggregateType),
		snap.AggregateID,
		snap.Version.Int(),
		snap.State,
		snap.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save snapshot for %s %s: %w", snap.AggregateType, snap.AggregateID, err)
	}

	return nil
}

// Load retrieves the latest snapshot for an aggregate.
// Returns ErrSnapshotNotFound if no snapshot exists.
func (s *SQLSnapshotStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	query := `SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = $1 AND aggregate_id = $2`

	var (
		version    int
		stateBytes []byte
		createdAt  time.Time
	)

	err := s.db.QueryRowContext(ctx, query, string(aggregateType), aggregateID).
		Scan(&version, &stateBytes, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, event.ErrSnapshotNotFound
		}

		return nil, fmt.Errorf("load snapshot for %s %s: %w", aggregateType, aggregateID, err)
	}

	return &event.Snapshot{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Version:       event.Version(version),
		State:         stateBytes,
		CreatedAt:     createdAt,
	}, nil
}

// LoadAtVersion retrieves a snapshot at or before a specific version.
// Returns ErrSnapshotNotFound if no snapshot exists or the stored version exceeds the requested version.
func (s *SQLSnapshotStore) LoadAtVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	query := `SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = $1 AND aggregate_id = $2`

	var (
		snapVersion int
		stateBytes  []byte
		createdAt   time.Time
	)

	err := s.db.QueryRowContext(
		ctx,
		query,
		string(aggregateType),
		aggregateID,
	).Scan(&snapVersion, &stateBytes, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, event.ErrSnapshotNotFound
		}

		return nil, fmt.Errorf("load snapshot at version %d for %s %s: %w",
			version, aggregateType, aggregateID, err)
	}

	if snapVersion > version.Int() {
		return nil, fmt.Errorf("load snapshot at version %d for %s %s: %w",
			version, aggregateType, aggregateID, event.ErrSnapshotNotFound)
	}

	return &event.Snapshot{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Version:       event.Version(snapVersion),
		State:         stateBytes,
		CreatedAt:     createdAt,
	}, nil
}

// Delete removes a snapshot for an aggregate.
func (s *SQLSnapshotStore) Delete(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	query := `DELETE FROM snapshots WHERE aggregate_type = $1 AND aggregate_id = $2`

	_, err := s.db.ExecContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return fmt.Errorf("delete snapshot for %s %s: %w", aggregateType, aggregateID, err)
	}

	return nil
}

var _ event.SnapshotStore = (*SQLSnapshotStore)(nil)
