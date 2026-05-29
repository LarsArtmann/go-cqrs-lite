package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/snapshot"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

type SQLSnapshotStore struct {
	sqlpkg.Base
}

func NewSQLSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

func NewSQLiteSnapshotStore(db *sql.DB) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

func NewSQLSnapshotStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLSnapshotStore, error) {
	return newSQLSnapshotStoreWithDialect(db, d)
}

func newSQLSnapshotStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLSnapshotStore, error) {
	base, err := sqlpkg.NewBase(db, d)
	if err != nil {
		return nil, err
	}
	return &SQLSnapshotStore{Base: base}, nil
}

func SnapshotSchema() string       { return sqlpkg.PostgresDialect{}.SnapshotSchema() }
func SQLiteSnapshotSchema() string { return sqlpkg.SQLiteDialect{}.SnapshotSchema() }

func (s *SQLSnapshotStore) Save(ctx context.Context, snap snapshot.Snapshot) error {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "snapshot.save", trace.SpanKindClient,
		trace.WithAttributes(append(cqrsotel.AggregateAttrs(snap.AggregateType, snap.AggregateID),
			attribute.Int(cqrsotel.AttrAggregateVersion, snap.Version.Int()))...))
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
		return event.WrapInfrastructure(err, "storage.save_snapshot",
			fmt.Sprintf("save snapshot for %s %s", snap.AggregateType, snap.AggregateID))
	}
	return nil
}

func (s *SQLSnapshotStore) Load(ctx context.Context, ref event.AggregateRef) (*snapshot.Snapshot, error) {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "snapshot.load", trace.SpanKindClient,
		trace.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...))
	defer span.End()
	snap, err := s.querySnapshot(ctx, ref)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, "storage.load_snapshot",
			fmt.Sprintf("load snapshot for %s %s", ref.Type, ref.ID))
	}
	return snap, nil
}

func (s *SQLSnapshotStore) LoadAtVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "snapshot.load_at_version", trace.SpanKindClient,
		trace.WithAttributes(append(cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			attribute.Int(cqrsotel.AttrAggregateVersion, version.Int()))...))
	defer span.End()
	snap, err := s.querySnapshot(ctx, ref)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, "storage.load_snapshot_version",
			fmt.Sprintf("load snapshot at version %d for %s %s", version, ref.Type, ref.ID))
	}
	if snap.Version.Cmp(version) > 0 {
		err := event.WrapRejection(snapshot.ErrSnapshotNotFound, "storage.snapshot_version_exceeded",
			fmt.Sprintf("load snapshot at version %d for %s %s", version, ref.Type, ref.ID))
		cqrsotel.RecordError(span, err)
		return nil, err
	}
	return snap, nil
}

func (s *SQLSnapshotStore) querySnapshot(ctx context.Context, ref event.AggregateRef) (*snapshot.Snapshot, error) {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	query := fmt.Sprintf(`SELECT version, state, created_at FROM `+sqlpkg.TableSnapshots+`
		WHERE aggregate_type = %s AND aggregate_id = %s`, p1, p2)
	return s.scanSnapshot(s.DB.QueryRowContext(ctx, query, string(ref.Type), ref.ID), ref)
}

func (s *SQLSnapshotStore) scanSnapshot(row *sql.Row, ref event.AggregateRef) (*snapshot.Snapshot, error) {
	var version int
	var stateBytes []byte
	timeDest := s.Dialect.ScanTimeDest()
	err := row.Scan(&version, &stateBytes, timeDest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, event.WrapRejection(snapshot.ErrSnapshotNotFound, "storage.snapshot_not_found",
				fmt.Sprintf("%s/%s at v%d", ref.Type, ref.ID, event.Version(version)))
		}
		return nil, event.WrapInfrastructure(err, "storage.scan_snapshot",
			fmt.Sprintf("scan snapshot for %s/%s", ref.Type, ref.ID))
	}
	createdAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_snapshot_created_at", "parse snapshot created_at")
	}
	return &snapshot.Snapshot{
		AggregateID: ref.ID, AggregateType: ref.Type,
		Version: event.Version(version), State: stateBytes, CreatedAt: createdAt,
	}, nil
}

func (s *SQLSnapshotStore) Delete(ctx context.Context, ref event.AggregateRef) error {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	return sqlpkg.DeleteByAggregate(s.DB, ctx, ref, sqlpkg.TableSnapshots, p1, p2, "snapshot")
}

var (
	_ snapshot.SnapshotSink   = (*SQLSnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*SQLSnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*SQLSnapshotStore)(nil)
)
