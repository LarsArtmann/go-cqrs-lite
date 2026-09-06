package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// MigrateSnapshotColumnsToStream renames the pre-v5 snapshots-table columns
// aggregate_type/aggregate_id to the honest stream vocabulary (stream_type/
// stream_id) — the 2026-08-22 T18 wire-tag audit; see
// docs/V5-MIGRATION-GUIDE.md §3. Rows move with their columns, so no data
// backfill is needed; a plain RENAME is metadata-only.
//
// The migration is idempotent and safe to run on every boot: it is a no-op
// when the snapshots table does not exist or already carries the stream
// columns. Every InitSchema helper calls it automatically; call it directly
// only when managing schema outside those helpers.
//
// ALTER TABLE ... RENAME COLUMN is supported by PostgreSQL, SQLite (>= 3.25),
// DuckDB, MySQL (>= 8.0), and MariaDB (>= 10.5).
func MigrateSnapshotColumnsToStream(ctx context.Context, db *sql.DB, d sqlpkg.Dialect) error {
	columns, err := probeTableColumns(ctx, db, d, sqlpkg.TableSnapshots)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"storage.snapshot_column_probe",
			"probe snapshots table columns for the v5 stream rename",
		)
	}

	if !containsString(columns, "aggregate_type") && !containsString(columns, "aggregate_id") {
		return nil
	}

	if containsString(columns, "stream_type") || containsString(columns, "stream_id") {
		return errorfamily.NewCorruption(
			"storage.snapshot_column_mixed",
			"snapshots table carries both aggregate and stream columns; "+
				"manual reconciliation required",
		)
	}

	for _, rename := range [][2]string{
		{"aggregate_type", "stream_type"},
		{"aggregate_id", "stream_id"},
	} {
		if !containsString(columns, rename[0]) {
			continue
		}

		stmt := fmt.Sprintf(
			"ALTER TABLE %s RENAME COLUMN %s TO %s",
			sqlpkg.TableSnapshots, rename[0], rename[1],
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"storage.snapshot_column_rename",
				"exec "+stmt,
			)
		}
	}

	return nil
}

// probeTableColumns lists the column names of table using the introspection
// the dialect supports: PRAGMA table_info on SQLite-family servers,
// information_schema.columns everywhere else.
func probeTableColumns(
	ctx context.Context,
	db *sql.DB,
	d sqlpkg.Dialect,
	table string,
) ([]string, error) {
	if _, ok := d.(sqlpkg.SQLiteDialect); ok {
		return probeSQLiteColumns(ctx, db, table)
	}

	_, isMySQL := d.(sqlpkg.MySQLDialect)

	return probeInformationSchemaColumns(ctx, db, table, isMySQL)
}

func probeSQLiteColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var (
			cid                int
			name, colType      string
			notNull            int
			defaultValue, pk   sql.NullString
		)

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, rows.Err()
}

func probeInformationSchemaColumns(
	ctx context.Context,
	db *sql.DB,
	table string,
	limitToCurrentSchema bool,
) ([]string, error) {
	query := "SELECT column_name FROM information_schema.columns WHERE table_name = ?"
	if limitToCurrentSchema {
		query += " AND table_schema = DATABASE()"
	}

	rows, err := db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, rows.Err()
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}

	return false
}
