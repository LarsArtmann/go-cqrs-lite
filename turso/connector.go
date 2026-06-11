package turso

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// DbPath is a phantom type for local database file paths.
type DbPath string

func (p DbPath) String() string { return string(p) }

func (p DbPath) IsZero() bool { return p == "" }

// RemoteURL is a phantom type for Turso remote server URLs.
type RemoteURL string

func (u RemoteURL) String() string { return string(u) }

func (u RemoteURL) IsZero() bool { return u == "" }

// AuthToken is a phantom type for Turso authentication tokens.
type AuthToken string

func (t AuthToken) String() string { return string(t) }

func (t AuthToken) IsZero() bool { return t == "" }

// Open opens a local Turso database file and returns a *sql.DB
// compatible with all SQLite* adapters in this package.
//
// The caller is responsible for closing the returned *sql.DB.
func Open(dbPath DbPath) (*sql.DB, error) {
	database, err := sql.Open("turso", string(dbPath))
	if err != nil {
		return nil, event.WrapInfrastructure(err, "turso.open",
			"open turso database at "+string(dbPath))
	}

	return database, nil
}

// OpenInMemory opens an in-memory Turso database and returns a *sql.DB.
// Useful for testing and development.
func OpenInMemory() (*sql.DB, error) {
	return Open(":memory:")
}

// InitSchema creates all required tables in a Turso database.
// Turso uses the same DDL as SQLite.
func InitSchema(ctx context.Context, db *sql.DB) error {
	return storage.SQLiteInitSchema(ctx, db) //nolint:wrapcheck // transparent delegation to storage
}

// NewEventStore creates an event store backed by a Turso database.
// Delegates to NewSQLiteEventStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewEventStore(db *sql.DB) (*storage.SQLEventStore, error) {
	return storage.NewSQLiteEventStore(db) //nolint:wrapcheck // transparent delegation to storage
}

// NewSnapshotStore creates a snapshot store backed by a Turso database.
// Delegates to NewSQLiteSnapshotStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewSnapshotStore(db *sql.DB) (*storage.SQLSnapshotStore, error) {
	return storage.NewSQLiteSnapshotStore( //nolint:wrapcheck // transparent delegation to storage
		db,
	)
}

// NewCheckpointStore creates a checkpoint store backed by a Turso database.
// Delegates to NewSQLiteCheckpointStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewCheckpointStore(db *sql.DB) (*storage.SQLCheckpointStore, error) {
	return storage.NewSQLiteCheckpointStore( //nolint:wrapcheck // transparent delegation to storage
		db,
	)
}

//nolint:gochecknoglobals // backward-compatible aliases
var (
	OpenTurso               = Open
	OpenTursoInMemory       = OpenInMemory
	TursoInitSchema         = InitSchema
	NewTursoEventStore      = NewEventStore
	NewTursoSnapshotStore   = NewSnapshotStore
	NewTursoCheckpointStore = NewCheckpointStore
)
