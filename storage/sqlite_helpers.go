package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func parseSQLiteTimestamp(s string) (time.Time, error) {
	return sqlpkg.ParseSQLiteTimestamp(s)
}

func OpenSQLite(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_loc=auto&_time_format=sqlite")
	if err != nil {
		return nil, event.WrapInfrastructure(
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
			return event.WrapInfrastructure(err, "storage.exec_ddl", "exec DDL: "+ddl)
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
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return event.WrapInfrastructure(err, "storage.enable_wal", "exec "+pragma)
		}
	}
	return nil
}

// SQLiteEnableForeignKeys turns on SQLite foreign-key enforcement for the
// given connection. This is opt-in (not enabled by default) because existing
// databases may contain orphaned references that would cause errors once
// enforcement is active. Call after opening the database if referential
// integrity is required.
func SQLiteEnableForeignKeys(ctx context.Context, db *sql.DB) error {
	const pragma = "PRAGMA foreign_keys=ON"

	if _, err := db.ExecContext(ctx, pragma); err != nil {
		return event.WrapInfrastructure(err, "storage.enable_foreign_keys", "exec "+pragma)
	}

	return nil
}

// SQLiteApplyOptimizations sets performance PRAGMAs recommended for CQRS
// workloads: cache_size (64 MB page cache), temp_store=MEMORY, and
// mmap_size=256 MB. These are safe, portable SQLite settings that improve
// throughput without durability trade-offs. Call after schema creation.
func SQLiteApplyOptimizations(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA cache_size=-65536",   // 64 MB page cache
		"PRAGMA temp_store=MEMORY",   // avoid temp files on disk
		"PRAGMA mmap_size=268435456", // 256 MB memory-mapped I/O
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return event.WrapInfrastructure(err, "storage.apply_optimizations", "exec "+pragma)
		}
	}
	return nil
}

func ConfigureSQLitePool(db *sql.DB) { db.SetMaxOpenConns(1) }
func ConfigureTursoPool(db *sql.DB)  { db.SetMaxOpenConns(1) }

func PostgresInitSchema(ctx context.Context, db *sql.DB) error {
	return execDDL(ctx, db, []string{sqlpkg.PostgresSchemaEmbed()})
}
