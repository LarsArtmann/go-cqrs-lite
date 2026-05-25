package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLCheckpointStore persists projection checkpoints in a SQL database.
type SQLCheckpointStore struct {
	db      *sql.DB
	dialect Dialect
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
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLCheckpointStore{db: db, dialect: d}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLCheckpointStore) Close() error { return nil }

// CheckpointSchema returns the SQL DDL for creating the checkpoints table.
func CheckpointSchema() string { return PostgresDialect{}.CheckpointSchema() }

// SQLiteCheckpointSchema returns the SQL DDL for creating the checkpoints table (SQLite variant).
func SQLiteCheckpointSchema() string { return SQLiteDialect{}.CheckpointSchema() }

// Load returns the last processed event ID for a projection.
func (s *SQLCheckpointStore) Load(ctx context.Context, projectionName string) (id.EventID, error) {
	return sharedCheckpointLoad(ctx, s.db, projectionName, s.dialect)
}

// Save persists the last processed event ID for a projection.
func (s *SQLCheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	eventID id.EventID,
) error {
	return sharedCheckpointSave(ctx, s.db, projectionName, eventID, s.dialect)
}

var _ event.CheckpointStore = (*SQLCheckpointStore)(nil)
