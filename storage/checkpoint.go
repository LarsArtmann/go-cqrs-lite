package storage

import (
	"context"
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

type SQLCheckpointStore struct {
	sqlpkg.DBHandle
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
	handle, err := sqlpkg.NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}
	return &SQLCheckpointStore{DBHandle: handle}, nil
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
		return event.Checkpoint{}, errorfamily.WrapInfrastructure(err,
			"storage.load_checkpoint",
			"load checkpoint for projection "+projectionName)
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
		return errorfamily.WrapInfrastructure(err,
			"storage.save_checkpoint",
			"save checkpoint for projection "+projectionName)
	}
	return nil
}

func (s *SQLCheckpointStore) startSpan(
	ctx context.Context,
	name, projectionName string,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), name, cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrString(cqrsotel.AttrProjectionName, projectionName)))
}

var (
	_ event.CheckpointSink   = (*SQLCheckpointStore)(nil)
	_ event.CheckpointSource = (*SQLCheckpointStore)(nil)
	_ event.CheckpointStore  = (*SQLCheckpointStore)(nil)
)
