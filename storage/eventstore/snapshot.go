package eventstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SQLSnapshotStore implements snapshot.SnapshotStore backed by a SQL database.
type SQLSnapshotStore struct {
	sqlpkg.DBHandle
}

// NewSQLSnapshotStore creates a new SQLSnapshotStore using the PostgreSQL dialect.
func NewSQLSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteSnapshotStore creates a new SQLSnapshotStore using the SQLite dialect.
func NewSQLiteSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLSnapshotStoreWithDialect creates a new SQLSnapshotStore with the given SQL dialect.
func NewSQLSnapshotStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, d)
}

func newSQLSnapshotStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLSnapshotStore, error) {
	handle, err := sqlpkg.NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}
	return &SQLSnapshotStore{DBHandle: handle}, nil
}

// SnapshotSchema returns the PostgreSQL DDL for the snapshots table.
func SnapshotSchema() string { return sqlpkg.PostgresDialect{}.SnapshotSchema() }

// SQLiteSnapshotSchema returns the SQLite DDL for the snapshots table.
func SQLiteSnapshotSchema() string { return sqlpkg.SQLiteDialect{}.SnapshotSchema() }

func (s *SQLSnapshotStore) Save(ctx context.Context, snap snapshot.Snapshot) error {
	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"snapshot.save",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.AggregateAttrs(snap.AggregateType, snap.AggregateID),
				cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, snap.Version.Int()))...,
		),
	)
	defer span.End()
	p1, p2, p3, p4, p5 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2),
		s.Dialect.Placeholder(3), s.Dialect.Placeholder(4), s.Dialect.Placeholder(5)
	query := fmt.Sprintf(
		`INSERT INTO `+sqlpkg.TableSnapshots+` (aggregate_type, aggregate_id, version, state, created_at)
		VALUES (%s, %s, %s, %s, %s)
		ON CONFLICT (aggregate_type, aggregate_id)
		DO UPDATE SET version = EXCLUDED.version, state = EXCLUDED.state, created_at = EXCLUDED.created_at`,
		p1,
		p2,
		p3,
		p4,
		p5,
	)
	_, err := s.DB.ExecContext(ctx, query, string(snap.AggregateType), snap.AggregateID,
		snap.Version.Int(), snap.State, s.Dialect.FormatTime(snap.CreatedAt))
	if err != nil {
		cqrsotel.RecordError(span, err)
		return errorfamily.WrapInfrastructure(err, "storage.save_snapshot",
			fmt.Sprintf("save snapshot for %s %s", snap.AggregateType, snap.AggregateID))
	}
	return nil
}

func (s *SQLSnapshotStore) Load(
	ctx context.Context,
	ref id.AggregateRef,
) (*snapshot.Snapshot, error) {
	ctx, span := sqlpkg.StartAggregateSpan(ctx, "snapshot.load", ref)
	defer span.End()
	snap, err := s.querySnapshot(ctx, ref)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.load_snapshot",
			fmt.Sprintf("load snapshot for %s %s", ref.Type, ref.ID))
	}
	return snap, nil
}

func (s *SQLSnapshotStore) LoadAtVersion(
	ctx context.Context,
	ref id.AggregateRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"snapshot.load_at_version",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, version.Int()))...),
	)
	defer span.End()
	snap, err := s.querySnapshotAtVersion(ctx, ref, version)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.load_snapshot_version",
			fmt.Sprintf("load snapshot at version %d for %s %s", version, ref.Type, ref.ID))
	}
	return snap, nil
}

func (s *SQLSnapshotStore) querySnapshotAtVersion(
	ctx context.Context,
	ref id.AggregateRef,
	maxVersion event.Version,
) (*snapshot.Snapshot, error) {
	p1, p2, p3 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2), s.Dialect.Placeholder(3)
	query := fmt.Sprintf(`SELECT version, state, created_at FROM `+sqlpkg.TableSnapshots+`
		WHERE aggregate_type = %s AND aggregate_id = %s AND version <= %s
		ORDER BY version DESC LIMIT 1`, p1, p2, p3)
	return s.scanSnapshot(
		s.DB.QueryRowContext(ctx, query, string(ref.Type), ref.ID, maxVersion.Int()),
		ref,
	)
}

func (s *SQLSnapshotStore) querySnapshot(
	ctx context.Context,
	ref id.AggregateRef,
) (*snapshot.Snapshot, error) {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	query := fmt.Sprintf(`SELECT version, state, created_at FROM `+sqlpkg.TableSnapshots+`
		WHERE aggregate_type = %s AND aggregate_id = %s`, p1, p2)
	return s.scanSnapshot(s.DB.QueryRowContext(ctx, query, string(ref.Type), ref.ID), ref)
}

func (s *SQLSnapshotStore) scanSnapshot(
	row *sql.Row,
	ref id.AggregateRef,
) (*snapshot.Snapshot, error) {
	var version int
	var stateBytes []byte
	timeDest := s.Dialect.ScanTimeDest()
	err := row.Scan(&version, &stateBytes, timeDest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorfamily.WrapRejection(
				snapshot.ErrSnapshotNotFound,
				"storage.snapshot_not_found",
				fmt.Sprintf("%s/%s at v%d", ref.Type, ref.ID, event.Version(version)),
			)
		}
		return nil, errorfamily.WrapInfrastructure(err, "storage.scan_snapshot",
			fmt.Sprintf("scan snapshot for %s/%s", ref.Type, ref.ID))
	}
	createdAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"storage.parse_snapshot_created_at",
			"parse snapshot created_at",
		)
	}
	return &snapshot.Snapshot{
		AggregateID: ref.ID, AggregateType: ref.Type,
		Version: event.Version(version), State: stateBytes, CreatedAt: createdAt,
	}, nil
}

func (s *SQLSnapshotStore) Delete(ctx context.Context, ref id.AggregateRef) error {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	return sqlpkg.DeleteByAggregate(s.DB, ctx, ref, sqlpkg.TableSnapshots, p1, p2, "snapshot")
}

var (
	_ snapshot.SnapshotSink   = (*SQLSnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*SQLSnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*SQLSnapshotStore)(nil)
)
