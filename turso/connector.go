package turso

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/storage"
)

// OpenTurso opens a local Turso database file and returns a *sql.DB
// compatible with all SQLite* adapters in this package.
//
// The caller is responsible for closing the returned *sql.DB.
func OpenTurso(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("turso", dbPath)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "turso.open",
			"open turso database at "+dbPath)
	}

	return db, nil
}

// OpenTursoInMemory opens an in-memory Turso database and returns a *sql.DB.
// Useful for testing and development.
func OpenTursoInMemory() (*sql.DB, error) {
	return OpenTurso(":memory:")
}

// TursoInitSchema creates all required tables in a Turso database.
// Turso uses the same DDL as SQLite.
func TursoInitSchema(ctx context.Context, db *sql.DB) error {
	return storage.SQLiteInitSchema(ctx, db)
}

// NewTursoEventStore creates an event store backed by a Turso database.
// Delegates to NewSQLiteEventStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewTursoEventStore(db *sql.DB) (*storage.SQLEventStore, error) {
	return storage.NewSQLiteEventStore(db)
}

// NewTursoSnapshotStore creates a snapshot store backed by a Turso database.
// Delegates to NewSQLiteSnapshotStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewTursoSnapshotStore(db *sql.DB) (*storage.SQLSnapshotStore, error) {
	return storage.NewSQLiteSnapshotStore(db)
}

// NewTursoOutbox creates an outbox backed by a Turso database.
// Delegates to NewSQLiteOutbox — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewTursoOutbox(db *sql.DB) (*storage.SQLOutbox, error) {
	return storage.NewSQLiteOutbox(db)
}

// NewTursoCheckpointStore creates a checkpoint store backed by a Turso database.
// Delegates to NewSQLiteCheckpointStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewTursoCheckpointStore(db *sql.DB) (*storage.SQLCheckpointStore, error) {
	return storage.NewSQLiteCheckpointStore(db)
}

// NewTursoSagaStore creates a saga state store backed by a Turso database.
// Delegates to NewSQLiteSagaStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewTursoSagaStore(db *sql.DB) (*storage.SQLSagaStore, error) {
	return storage.NewSQLiteSagaStore(db)
}

// NewTursoTransactionalStore creates a transactional store backed by a Turso database.
// Delegates to NewSQLTransactionalStore — Turso uses the same SQL dialect as SQLite.
// Combines event persistence and outbox append in a single transaction.
func NewTursoTransactionalStore(
	store *storage.SQLEventStore,
	outbox *storage.SQLOutbox,
) (*storage.SQLTransactionalStore, error) {
	return storage.NewSQLTransactionalStore(store, outbox)
}
