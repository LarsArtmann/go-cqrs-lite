package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// SQLSnapshotStore persists aggregate snapshots in a SQL database.
type SQLSnapshotStore struct {
	sqlBase
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

// NewSQLSnapshotStoreWithDialect creates a new SQL-backed snapshot store with a custom dialect.
// This enables consumers to use any SQL backend by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLSnapshotStoreWithDialect(db *sql.DB, d Dialect) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, d)
}

func newSQLSnapshotStoreWithDialect(db *sql.DB, d Dialect) (*SQLSnapshotStore, error) {
	base, err := newSQLBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLSnapshotStore{sqlBase: base}, nil
}

// SnapshotSchema returns the SQL DDL for creating the snapshots table.
func SnapshotSchema() string { return PostgresDialect{}.SnapshotSchema() }

// SQLiteSnapshotSchema returns the SQL DDL for creating the snapshots table (SQLite variant).
func SQLiteSnapshotSchema() string { return SQLiteDialect{}.SnapshotSchema() }

// Save persists a snapshot for an aggregate.
// State is stored as-is ([]byte) — no additional marshaling is applied.
func (s *SQLSnapshotStore) Save(ctx context.Context, snap event.Snapshot) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "snapshot.save",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrsWithVersion(
			string(snap.AggregateType), snap.AggregateID.String(), snap.Version.Int(),
		)...),
	)
	defer span.End()

	p1, p2, p3, p4, p5 := s.dialect.Placeholder(1), s.dialect.Placeholder(2),
		s.dialect.Placeholder(3), s.dialect.Placeholder(4), s.dialect.Placeholder(5)

	query := fmt.Sprintf(
		`INSERT INTO `+tableSnapshots+` (aggregate_type, aggregate_id, version, state, created_at)
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
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.save_snapshot",
			fmt.Sprintf("save snapshot for %s %s", snap.AggregateType, snap.AggregateID))
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
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "snapshot.load",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrs(string(aggregateType), aggregateID.String())...),
	)
	defer span.End()

	snap, err := s.querySnapshot(ctx, aggregateType, aggregateID)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.load_snapshot",
			fmt.Sprintf("load snapshot for %s %s", aggregateType, aggregateID))
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
		return nil, event.WrapInfrastructure(
			err,
			"storage.load_snapshot_version",
			fmt.Sprintf(
				"load snapshot at version %d for %s %s",
				version,
				aggregateType,
				aggregateID,
			),
		)
	}

	if snap.Version.Cmp(version) > 0 {
		return nil, event.WrapRejection(
			event.ErrSnapshotNotFound,
			"storage.snapshot_version_exceeded",
			fmt.Sprintf(
				"load snapshot at version %d for %s %s",
				version,
				aggregateType,
				aggregateID,
			),
		)
	}

	return snap, nil
}

func (s *SQLSnapshotStore) querySnapshot(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(`SELECT version, state, created_at FROM `+tableSnapshots+`
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
			return nil, event.WrapRejection(event.ErrSnapshotNotFound, "storage.snapshot_not_found",
				fmt.Sprintf("%s/%s at v%d", aggregateType, aggregateID, event.Version(version)))
		}

		return nil, event.WrapInfrastructure(err, "storage.scan_snapshot",
			fmt.Sprintf("scan snapshot for %s/%s", aggregateType, aggregateID))
	}

	createdAt, err := s.dialect.ParseTime(timeDest)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_snapshot_created_at",
			"parse snapshot created_at")
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
		tableSnapshots, p1, p2, "snapshot",
	)
}

var _ event.SnapshotStore = (*SQLSnapshotStore)(nil)
