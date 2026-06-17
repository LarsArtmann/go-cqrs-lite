package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
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
	return execDDL(ctx, db, []string{
		sqlpkg.SQLiteSchema(),
		sqlpkg.SQLiteDialect{}.CommandSchema(),
		sqlpkg.SQLiteDialect{}.QuerySchema(),
		sqlpkg.SQLiteDialect{}.SnapshotSchema(),
		sqlpkg.SQLiteDialect{}.CheckpointSchema(),
	})
}

func SQLiteEnableWAL(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
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

func ConfigureSQLitePool(db *sql.DB) { db.SetMaxOpenConns(1) }
func ConfigureTursoPool(db *sql.DB)  { db.SetMaxOpenConns(1) }

func PostgresInitSchema(ctx context.Context, db *sql.DB) error {
	pg := sqlpkg.PostgresDialect{}
	return execDDL(
		ctx,
		db,
		[]string{
			pg.EventSchema(),
			pg.CommandSchema(),
			pg.QuerySchema(),
			pg.SnapshotSchema(),
			pg.CheckpointSchema(),
		},
	)
}
