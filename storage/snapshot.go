package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLSnapshotStore persists aggregate snapshots in a SQL database.
type SQLSnapshotStore struct {
	db      *sql.DB
	dialect Dialect
}

// NewSQLSnapshotStore creates a new SQL-backed snapshot store using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, PostgresDialect{})
}

// NewSQLiteSnapshotStore creates a new SQLite-backed snapshot store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, SQLiteDialect{})
}

func newSQLSnapshotStoreWithDialect(db *sql.DB, d Dialect) (*SQLSnapshotStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLSnapshotStore{db: db, dialect: d}, nil
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
// State is stored as-is ([]byte) — no additional marshaling is applied.
func (s *SQLSnapshotStore) Save(ctx context.Context, snap event.Snapshot) error {
	p1, p2, p3, p4, p5 := s.dialect.Placeholder(1), s.dialect.Placeholder(2),
		s.dialect.Placeholder(3), s.dialect.Placeholder(4), s.dialect.Placeholder(5)

	query := fmt.Sprintf(
		`INSERT INTO snapshots (aggregate_type, aggregate_id, version, state, created_at)
		VALUES (%s, %s, %s, %s, %s)
		ON CONFLICT (aggregate_type, aggregate_id)
		DO UPDATE SET version = EXCLUDED.version, state = EXCLUDED.state, created_at = EXCLUDED.created_at`,
		p1,
		p2,
		p3,
		p4,
		p5,
	)

	_, err := s.db.ExecContext(
		ctx,
		query,
		string(snap.AggregateType),
		snap.AggregateID,
		snap.Version.Int(),
		snap.State,
		s.dialect.FormatTime(snap.CreatedAt),
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
	snap, err := s.querySnapshot(ctx, aggregateType, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("load snapshot for %s %s: %w", aggregateType, aggregateID, err)
	}

	return snap, nil
}

// LoadAtVersion retrieves a snapshot at or before a specific version.
// Returns ErrSnapshotNotFound if no snapshot exists or the stored version exceeds the requested version.
func (s *SQLSnapshotStore) LoadAtVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) (*event.Snapshot, error) {
	snap, err := s.querySnapshot(ctx, aggregateType, aggregateID)
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

func (s *SQLSnapshotStore) querySnapshot(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(`SELECT version, state, created_at FROM snapshots
		WHERE aggregate_type = %s AND aggregate_id = %s`, p1, p2)

	return s.scanSnapshot(
		s.db.QueryRowContext(ctx, query, string(aggregateType), aggregateID),
		aggregateType,
		aggregateID,
	)
}

func (s *SQLSnapshotStore) scanSnapshot(
	row *sql.Row,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	var (
		version    int
		stateBytes []byte
	)

	timeDest := s.dialect.ScanTimeDest()

	err := row.Scan(&version, &stateBytes, timeDest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"%s/%s: %w", aggregateType, aggregateID, event.ErrSnapshotNotFound,
			)
		}

		return nil, fmt.Errorf("scan snapshot for %s/%s: %w", aggregateType, aggregateID, err)
	}

	createdAt, err := s.dialect.ParseTime(timeDest)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	return &event.Snapshot{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       event.Version(version),
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
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	return deleteByAggregate(
		s.db, ctx, aggregateType, aggregateID,
		"snapshots", p1, p2, "snapshot",
	)
}

var _ event.SnapshotStore = (*SQLSnapshotStore)(nil)
