package storage

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

type SQLCheckpointStore struct {
	sqlpkg.Base
}

func NewSQLCheckpointStore(db *sql.DB) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

func NewSQLiteCheckpointStore(db *sql.DB) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

func NewSQLCheckpointStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, d)
}

func newSQLCheckpointStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCheckpointStore, error) {
	base, err := sqlpkg.NewBase(db, d)
	if err != nil {
		return nil, err
	}
	return &SQLCheckpointStore{Base: base}, nil
}

func CheckpointSchema() string       { return sqlpkg.PostgresDialect{}.CheckpointSchema() }
func SQLiteCheckpointSchema() string { return sqlpkg.SQLiteDialect{}.CheckpointSchema() }

func (s *SQLCheckpointStore) Load(
	ctx context.Context,
	projectionName string,
) (event.Checkpoint, error) {
	ctx, span := s.startSpan(ctx, "checkpoint.load", projectionName)
	defer span.End()
	cp, err := sqlpkg.SharedCheckpointLoad(ctx, s.DB, projectionName, s.Dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return event.Checkpoint{}, event.WrapInfrastructure(err,
			"storage.load_checkpoint",
			fmt.Sprintf("load checkpoint for projection %s", projectionName))
	}
	return cp, nil
}

func (s *SQLCheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	cp event.Checkpoint,
) error {
	ctx, span := s.startSpan(ctx, "checkpoint.save", projectionName)
	defer span.End()
	err := sqlpkg.SharedCheckpointSave(ctx, s.DB, projectionName, cp, s.Dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return event.WrapInfrastructure(err,
			"storage.save_checkpoint",
			fmt.Sprintf("save checkpoint for projection %s", projectionName))
	}
	return nil
}

func (s *SQLCheckpointStore) startSpan(
	ctx context.Context,
	name, projectionName string,
) (context.Context, trace.Span) {
	return cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), name, trace.SpanKindClient,
		trace.WithAttributes(attribute.String(cqrsotel.AttrProjectionName, projectionName)))
}

var (
	_ event.CheckpointSink   = (*SQLCheckpointStore)(nil)
	_ event.CheckpointSource = (*SQLCheckpointStore)(nil)
	_ event.CheckpointStore  = (*SQLCheckpointStore)(nil)
)
