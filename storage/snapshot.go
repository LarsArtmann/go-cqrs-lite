package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	json "github.com/go-json-experiment/json"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLSnapshotStore persists aggregate snapshots in a SQL database.
type SQLSnapshotStore struct {
	db *sql.DB
}

// NewSQLSnapshotStore creates a new SQL-backed snapshot store.
func NewSQLSnapshotStore(db *sql.DB) *SQLSnapshotStore {
	return &SQLSnapshotStore{db: db}
}

// Close releases the underlying database connection.
// Caller must not use the *sql.DB passed to NewSQLSnapshotStore after calling Close.
func (s *SQLSnapshotStore) Close() error {
	return errors.Wrap(s.db.Close(), "close snapshot store database connection")
}

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
func (s *SQLSnapshotStore) Save(ctx context.Context, snap event.Snapshot) error {
	state, err := json.Marshal(snap.State)
	if err != nil {
		return errors.Wrap(err, "marshal snapshot state")
	}

	query := `INSERT INTO snapshots (aggregate_type, aggregate_id, version, state, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (aggregate_type, aggregate_id)
		DO UPDATE SET version = EXCLUDED.version, state = EXCLUDED.state, created_at = EXCLUDED.created_at`

	_, err = s.db.ExecContext(
		ctx,
		query,
		string(snap.AggregateType),
		snap.AggregateID,
		snap.Version,
		state,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save snapshot for %s %s: %w", snap.AggregateType, snap.AggregateID, err)
	}

	return nil
}

// Load retrieves the latest snapshot for an aggregate.
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

// LoadAtVersion retrieves a snapshot at a specific version.
func (s *SQLSnapshotStore) LoadAtVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	query := `SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version = $3`

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
		version.Int(),
	).Scan(&snapVersion, &stateBytes, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("load snapshot at version %d for %s %s: %w",
			version, aggregateType, aggregateID, err)
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
