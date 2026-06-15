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
	_, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	if err != nil {
		return event.WrapInfrastructure(err, "storage.enable_wal", "enable WAL mode")
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
