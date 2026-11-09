//go:build integration

package storage

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"sync/atomic"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// mysqlTestCounter serializes the shared snapshots table on cqrs_test (the
// cqrs user has no CREATE DATABASE privilege; each run rebuilds the table).
var mysqlTestCounter atomic.Int32

func openMySQLForMigration(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — run against the userspace MariaDB " +
			"(see AGENTS.md Testing section)")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mysqlTestCounter.Add(1)

	if _, err := db.Exec("DROP TABLE IF EXISTS snapshots"); err != nil {
		t.Fatalf("drop stale snapshots table: %v", err)
	}

	return db
}

// TestMigrateSnapshotColumnsToStream_MariaDB runs the stream-column rename
// on live MariaDB (FOR UPDATE SKIP LOCKED dialect family): create the legacy
// snapshots table with aggregate columns, insert a row, migrate, and prove
// the row survived with its identity under the new columns.
func TestMigrateSnapshotColumnsToStream_MariaDB(t *testing.T) {
	db := openMySQLForMigration(t)
	ctx := context.Background()

	const legacyDDL = `CREATE TABLE snapshots (
		aggregate_type VARCHAR(255) NOT NULL,
		aggregate_id   VARCHAR(255) NOT NULL,
		version        BIGINT NOT NULL,
		state          LONGBLOB NOT NULL,
		created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (aggregate_type, aggregate_id)
	) ENGINE=InnoDB`
	if _, err := db.Exec(legacyDDL); err != nil {
		t.Fatalf("create legacy snapshots table: %v", err)
	}

	const insert = `INSERT INTO snapshots
		(aggregate_type, aggregate_id, version, state)
		VALUES ('User', '01HK1540X0841Y0A6BSX1VKR95', 7, ?)`
	if _, err := db.Exec(insert, []byte("state-bytes")); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.MySQLDialect{}); err != nil {
		t.Fatalf("migrate on MariaDB: %v", err)
	}

	// The migration uses information_schema probing on this dialect —
	// prove the rename through the probe itself.
	columns, err := probeInformationSchemaColumns(ctx, db, sqlpkg.MySQLDialect{},
		sqlpkg.TableSnapshots, true)
	if err != nil {
		t.Fatalf("probe columns: %v", err)
	}

	if !slices.Contains(columns, "stream_type") || !slices.Contains(columns, "stream_id") {
		t.Fatalf("stream columns missing after migrate: %v", columns)
	}

	if slices.Contains(columns, "aggregate_type") || slices.Contains(columns, "aggregate_id") {
		t.Fatalf("aggregate columns still present after migrate: %v", columns)
	}

	var streamType, streamID string

	var state []byte

	var version int

	err = db.QueryRow(`SELECT stream_type, stream_id, version, state
		FROM snapshots WHERE stream_id = ?`, "01HK1540X0841Y0A6BSX1VKR95").
		Scan(&streamType, &streamID, &version, &state)
	if err != nil {
		t.Fatalf("select migrated row: %v", err)
	}

	if streamType != "User" || streamID != "01HK1540X0841Y0A6BSX1VKR95" ||
		version != 7 || string(state) != "state-bytes" {
		t.Fatalf("row identity mismatch after migrate: %s/%s v%d %q",
			streamType, streamID, version, state)
	}

	if err := MigrateSnapshotColumnsToStream(ctx, db, sqlpkg.MySQLDialect{}); err != nil {
		t.Fatalf("re-migrate on MariaDB (idempotency): %v", err)
	}
}
