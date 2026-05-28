package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func parseSQLiteTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, event.WrapCorruption(
		ErrUnsupportedTimestamp,
		"storage.unsupported_timestamp",
		"unsupported timestamp format: "+s,
	)
}

// OpenSQLite opens a SQLite database file and returns a *sql.DB.
// The caller is responsible for closing the returned *sql.DB.
//
// Uses modernc.org/sqlite (pure Go, no CGO). The DSN is configured with
// auto-location and SQLite timestamp format for consistent time handling.
func OpenSQLite(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_loc=auto&_time_format=sqlite")
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.open_sqlite",
			"open sqlite database at "+dbPath)
	}

	return db, nil
}

// OpenSQLiteInMemory opens an in-memory SQLite database and returns a *sql.DB.
// Useful for testing and development. Data is lost when the process exits.
func OpenSQLiteInMemory() (*sql.DB, error) {
	return OpenSQLite("file::memory:")
}

func execDDL(ctx context.Context, db *sql.DB, ddls []string) error {
	for _, ddl := range ddls {
		_, err := db.ExecContext(ctx, ddl)
		if err != nil {
			return event.WrapInfrastructure(err, "storage.exec_ddl",
				"exec DDL: "+ddl)
		}
	}

	return nil
}

// SQLiteInitSchema creates all required tables in the SQLite database.
func SQLiteInitSchema(ctx context.Context, db *sql.DB) error {
	return execDDL(
		ctx,
		db,
		[]string{
			SQLiteSchema(),
			SQLiteSnapshotSchema(),
			SQLiteCheckpointSchema(),
		},
	)
}

// SQLiteEnableWAL enables Write-Ahead Logging for better concurrent read
// performance. WAL mode allows readers and a single writer to operate
// simultaneously without blocking each other.
//
// Must be called before any writes. Not compatible with shared-cache
// connections across multiple processes.
func SQLiteEnableWAL(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	if err != nil {
		return event.WrapInfrastructure(err, "storage.enable_wal",
			"enable WAL mode")
	}

	return nil
}

// ConfigureSQLitePool sets connection pool limits suitable for SQLite.
// SQLite requires at most one writer at a time, so MaxOpenConns is set to 1
// to prevent "database is locked" errors under concurrent access.
func ConfigureSQLitePool(db *sql.DB) {
	db.SetMaxOpenConns(1)
}

// ConfigureTursoPool sets connection pool limits suitable for Turso.
// Turso supports concurrent reads but serializes writes through its own
// busy timeout mechanism. MaxOpenConns is set to 1 for safe write behavior.
func ConfigureTursoPool(db *sql.DB) {
	db.SetMaxOpenConns(1)
}

// PostgresInitSchema creates all required tables in a PostgreSQL database.
func PostgresInitSchema(ctx context.Context, db *sql.DB) error {
	pg := PostgresDialect{}

	return execDDL(
		ctx,
		db,
		[]string{pg.EventSchema(), pg.SnapshotSchema(), pg.CheckpointSchema(), pg.OutboxSchema()},
	)
}
