package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// EnsureSQLiteDSNBusyTimeout appends _pragma=busy_timeout(ms) to the DSN so
// every connection the pool creates has the busy_timeout set. PRAGMAs set via
// db.Exec only apply to the connection that executes them; if the pool evicts
// and recreates a connection, the new one would not inherit the PRAGMA, causing
// SQLITE_BUSY errors under concurrent access. DSN-level pragmas are applied by
// the modernc.org/sqlite driver on every new connection.
func EnsureSQLiteDSNBusyTimeout(dsn string, ms int) string {
	if strings.Contains(dsn, "busy_timeout") {
		return dsn
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}

	return fmt.Sprintf("%s%s_pragma=busy_timeout(%d)", dsn, sep, ms)
}

func OpenSQLite(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_loc=auto&_time_format=sqlite")
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.open_sqlite",
			"open sqlite database at "+dbPath,
		)
	}
	return db, nil
}

func OpenSQLiteInMemory() (*sql.DB, error) { return OpenSQLite("file::memory:") }

func execDDL(ctx context.Context, db *sql.DB, ddls []string) error {
	for _, ddl := range ddls {
		_, err := db.ExecContext(ctx, ddl)
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "storage.exec_ddl", "exec DDL: "+ddl)
		}
	}
	return nil
}

// execPragmas runs each PRAGMA statement against db, wrapping the first failure
// with errCode. Shared by SQLiteEnableWAL and SQLiteApplyOptimizations.
func execPragmas(ctx context.Context, db *sql.DB, pragmas []string, errCode string) error {
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return errorfamily.WrapInfrastructure(err, errCode, "exec "+pragma)
		}
	}
	return nil
}

func SQLiteInitSchema(ctx context.Context, db *sql.DB) error {
	return execDDL(ctx, db, []string{sqlpkg.SQLiteSchemaEmbed()})
}

// SQLiteEnableWAL enables Write-Ahead Logging for better read concurrency and
// configures production-safe pragmas: synchronous=NORMAL (safe with WAL, avoids
// an fsync per transaction — 3-10x faster than FULL) and busy_timeout=5000
// (eliminates "database is locked" errors under concurrency).
func SQLiteEnableWAL(ctx context.Context, db *sql.DB) error {
	return execPragmas(ctx, db, []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}, "storage.enable_wal")
}

// SQLiteEnableForeignKeys turns on SQLite foreign-key enforcement for the
// given connection. This is opt-in (not enabled by default) because existing
// databases may contain orphaned references that would cause errors once
// enforcement is active. Call after opening the database if referential
// integrity is required.
func SQLiteEnableForeignKeys(ctx context.Context, db *sql.DB) error {
	const pragma = "PRAGMA foreign_keys=ON"

	if _, err := db.ExecContext(ctx, pragma); err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.enable_foreign_keys", "exec "+pragma)
	}

	return nil
}

// SQLiteApplyOptimizations sets performance PRAGMAs recommended for CQRS
// workloads: cache_size (64 MB page cache), temp_store=MEMORY, and
// mmap_size=256 MB. These are safe, portable SQLite settings that improve
// throughput without durability trade-offs. Call after schema creation.
func SQLiteApplyOptimizations(ctx context.Context, db *sql.DB) error {
	return execPragmas(ctx, db, []string{
		"PRAGMA cache_size=-65536",   // 64 MB page cache
		"PRAGMA temp_store=MEMORY",   // avoid temp files on disk
		"PRAGMA mmap_size=268435456", // 256 MB memory-mapped I/O
	}, "storage.apply_optimizations")
}

// ConfigureSQLitePool caps the connection pool at 1 for SQLite — WAL mode
// serializes writes regardless of pool size, and a single connection eliminates
// "database is locked" errors entirely.
func ConfigureSQLitePool(db *sql.DB) { db.SetMaxOpenConns(1) }

// ConfigureTursoPool caps the connection pool at 1 for the embedded Turso
// Database engine. The engine defaults to WAL mode, which — like SQLite WAL —
// serializes writes through a single exclusive write lock. The cap makes this
// explicit and prevents ErrTursoBusy under read+write contention.
//
// This does sacrifice read concurrency (WAL allows N concurrent readers), but
// for a library default, eliminating lock errors is the safer choice.
// Consumers needing read concurrency can call db.SetMaxOpenConns(N) directly.
//
// The Turso engine also has an experimental MVCC mode (PRAGMA journal_mode=mvcc
// + BEGIN CONCURRENT) that enables true concurrent writes with row-level
// conflict detection. When MVCC becomes stable, raising the pool limit would
// unlock real write parallelism. See TODO_LIST.md → "Turso MVCC concurrent-write
// support" for the unblock criteria.
func ConfigureTursoPool(db *sql.DB) { db.SetMaxOpenConns(1) }

func PostgresInitSchema(ctx context.Context, db *sql.DB) error {
	return execDDL(ctx, db, []string{sqlpkg.PostgresSchemaEmbed()})
}

func DuckDBInitSchema(ctx context.Context, db *sql.DB) error {
	return execDDL(ctx, db, []string{sqlpkg.DuckDBSchemaEmbed()})
}
