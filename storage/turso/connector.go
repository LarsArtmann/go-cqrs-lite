package turso

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
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
//
// Forces MaxOpenConns=1 because SQLite ":memory:" databases are per-connection —
// each new connection gets its own empty database. Without this, parallel tests
// intermittently see "no such table" when the pool hands out a fresh connection.
//
// For parallel test suites that need many simultaneous databases, prefer
// [OpenTemp] with per-test temp directories — the Turso Database engine
// has resource limits that ":memory:" can exhaust under heavy parallelism.
func OpenInMemory() (*sql.DB, error) {
	database, err := Open(":memory:")
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(1)

	return database, nil
}

// OpenTemp opens a file-backed Turso database in the given directory.
// If dir is empty, uses the OS temp directory. The caller is responsible for
// closing the returned *sql.DB. For tests, pair with t.TempDir().
//
// Prefer this over [OpenInMemory] for parallel test suites — file-backed
// databases don't exhaust the Turso Database engine's native in-memory resource pool.
func OpenTemp(dir string) (*sql.DB, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	path := filepath.Join(dir, fmt.Sprintf("cqrs-test-%d.db", time.Now().UnixNano()))

	return Open(DbPath(path))
}

// InitSchema creates all required tables in a Turso database.
// Turso uses the same DDL as SQLite.
func InitSchema(ctx context.Context, db *sql.DB) error {
	return storage.SQLiteInitSchema(ctx, db)
}

// NewEventStore creates an event store backed by a Turso database.
// Delegates to NewSQLiteEventStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewEventStore(db *sql.DB) (*storage.SQLEventStore, error) {
	return storage.NewSQLiteEventStore(db)
}

// NewSnapshotStore creates a snapshot store backed by a Turso database.
// Delegates to NewSQLiteSnapshotStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewSnapshotStore(db *sql.DB) (*storage.SQLSnapshotStore, error) {
	return storage.NewSQLiteSnapshotStore(
		db,
	)
}

// NewCheckpointStore creates a checkpoint store backed by a Turso database.
// Delegates to NewSQLiteCheckpointStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewCheckpointStore(db *sql.DB) (*storage.SQLCheckpointStore, error) {
	return storage.NewSQLiteCheckpointStore(
		db,
	)
}

// NewCommandStore creates a command audit store backed by a Turso database.
// Delegates to NewSQLiteCommandStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewCommandStore(db *sql.DB) (*storage.SQLCommandStore, error) {
	return storage.NewSQLiteCommandStore(
		db,
	)
}

// NewQueryStore creates a query audit store backed by a Turso database.
// Delegates to NewSQLiteQueryStore — Turso uses the same SQL dialect as SQLite.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
func NewQueryStore(db *sql.DB) (*storage.SQLQueryStore, error) {
	return storage.NewSQLiteQueryStore(
		db,
	)
}

// NewViewStore creates a view store backed by a Turso database with real,
// queryable columns. Delegates to NewSQLiteViewStore — Turso uses the same
// SQL dialect as SQLite. The *sql.DB is borrowed, not owned — the caller is
// responsible for closing it.
func NewViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper storage.ViewMapper[V],
	opts ...storage.ViewStoreOption,
) (*storage.SQLViewStore[V, K], error) {
	return storage.NewSQLiteViewStore[V, K](db, mapper, opts...)
}

// ConfigurePool sets connection-pool defaults recommended for embedded
// Turso databases. The embedded Turso Database serializes writes through a
// single connection, so the pool is capped at one open connection to avoid
// "database is locked" errors under concurrent load.
//
// Call once after [Open] or [OpenSync]:
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	turso.ConfigurePool(db)
//
// Delegates to [storage.ConfigureTursoPool].
func ConfigurePool(db *sql.DB) {
	storage.ConfigureTursoPool(db)
}

//nolint:gochecknoglobals // backward-compatible aliases
var (
	OpenTurso               = Open
	OpenTursoInMemory       = OpenInMemory
	TursoInitSchema         = InitSchema
	NewTursoEventStore      = NewEventStore
	NewTursoSnapshotStore   = NewSnapshotStore
	NewTursoCheckpointStore = NewCheckpointStore
	NewTursoCommandStore    = NewCommandStore
	NewTursoQueryStore      = NewQueryStore
)
