//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// TestMigrateSnapshotColumnsToStream_DuckDB runs the stream-column rename on
// a live embedded DuckDB (T18 migration-verification tail): legacy
// aggregate-column table with a row, migrate, prove the row survived with
// its identity under the new columns, and prove idempotency. Complements the
// SQLite unit legs and the MariaDB integration leg.
func TestMigrateSnapshotColumnsToStream_DuckDB(t *testing.T) {
	ctx := context.Background()

	db, err := storage.OpenDuckDBInMemory()
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const legacyDDL = `CREATE TABLE snapshots (
		aggregate_type VARCHAR NOT NULL,
		aggregate_id   VARCHAR NOT NULL,
		version        INTEGER NOT NULL,
		state          BLOB NOT NULL,
		created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (aggregate_type, aggregate_id)
	)`
	if _, err := db.Exec(legacyDDL); err != nil {
		t.Fatalf("create legacy snapshots table: %v", err)
	}

	const streamID = "01HK1540X0841Y0A6BSX1VKR95"

	const insert = `INSERT INTO snapshots
		(aggregate_type, aggregate_id, version, state)
		VALUES ('User', ?, 7, ?)`
	if _, err := db.Exec(insert, streamID, []byte("state-bytes")); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := storage.MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.DuckDBDialect{}); err != nil {
		t.Fatalf("migrate on DuckDB: %v", err)
	}

	columns := duckdbSnapshotColumns(t, db)

	if !slices.Contains(columns, "stream_type") || !slices.Contains(columns, "stream_id") {
		t.Fatalf("stream columns missing after migrate: %v", columns)
	}

	if slices.Contains(columns, "aggregate_type") || slices.Contains(columns, "aggregate_id") {
		t.Fatalf("aggregate columns still present after migrate: %v", columns)
	}

	var (
		gotType, gotID string
		version        int
		state          []byte
	)

	err = db.QueryRow(`SELECT stream_type, stream_id, version, state
		FROM snapshots WHERE stream_id = $1`, streamID).
		Scan(&gotType, &gotID, &version, &state)
	if err != nil {
		t.Fatalf("select migrated row: %v", err)
	}

	if gotType != "User" || gotID != streamID || version != 7 || string(state) != "state-bytes" {
		t.Fatalf("row identity mismatch after migrate: %s/%s v%d %q",
			gotType, gotID, version, state)
	}

	if err := storage.MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.DuckDBDialect{}); err != nil {
		t.Fatalf("re-migrate on DuckDB (idempotency): %v", err)
	}
}

// duckdbSnapshotColumns probes the snapshots table through the same
// information_schema view the dialect-agnostic migration uses.
func duckdbSnapshotColumns(t *testing.T, db *sql.DB) []string {
	t.Helper()

	rows, err := db.Query(
		`SELECT column_name FROM information_schema.columns WHERE table_name = ?`,
		sqlpkg.TableSnapshots,
	)
	if err != nil {
		t.Fatalf("probe snapshots columns: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var names []string

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan column name: %v", err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}

	return names
}
