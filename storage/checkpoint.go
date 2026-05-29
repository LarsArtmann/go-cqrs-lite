package storage

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// SQLCheckpointStore persists projection checkpoints in a SQL database.
type SQLCheckpointStore struct {
	sqlBase
}

// NewSQLCheckpointStore creates a new SQL-backed checkpoint store using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCheckpointStore(db *sql.DB) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, PostgresDialect{})
}

// NewSQLiteCheckpointStore creates a new SQLite-backed checkpoint store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteCheckpointStore(db *sql.DB) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, SQLiteDialect{})
}

// NewSQLCheckpointStoreWithDialect creates a new SQL-backed checkpoint store with a custom dialect.
// This enables consumers to use any SQL backend by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCheckpointStoreWithDialect(db *sql.DB, d Dialect) (*SQLCheckpointStore, error) {
	return newSQLCheckpointStoreWithDialect(db, d)
}

func newSQLCheckpointStoreWithDialect(db *sql.DB, d Dialect) (*SQLCheckpointStore, error) {
	base, err := newSQLBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLCheckpointStore{sqlBase: base}, nil
}

// CheckpointSchema returns the SQL DDL for creating the checkpoints table.
func CheckpointSchema() string { return PostgresDialect{}.CheckpointSchema() }

// SQLiteCheckpointSchema returns the SQL DDL for creating the checkpoints table (SQLite variant).
func SQLiteCheckpointSchema() string { return SQLiteDialect{}.CheckpointSchema() }

// Load returns the last processed event ID for a projection.
func (s *SQLCheckpointStore) Load(ctx context.Context, projectionName string) (id.EventID, error) {
	ctx, span := s.startSpan(ctx, "checkpoint.load", projectionName)
	defer span.End()

	eventID, err := sharedCheckpointLoad(ctx, s.db, projectionName, s.dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return eventID, fmt.Errorf("load checkpoint for projection %s: %w", projectionName, err)
	}

	return eventID, nil
}

// Save persists the last processed event ID for a projection.
func (s *SQLCheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	eventID id.EventID,
) error {
	ctx, span := s.startSpan(ctx, "checkpoint.save", projectionName)
	defer span.End()

	err := sharedCheckpointSave(ctx, s.db, projectionName, eventID, s.dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return fmt.Errorf("save checkpoint for projection %s: %w", projectionName, err)
	}

	return nil
}

func (s *SQLCheckpointStore) startSpan(ctx context.Context, name, projectionName string) (context.Context, trace.Span) {
	return cqrsotel.StartSpan(
		ctx, tracer(), name,
		trace.SpanKindClient,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrProjectionName, projectionName),
		),
	)
}

var _ event.CheckpointStore = (*SQLCheckpointStore)(nil)
