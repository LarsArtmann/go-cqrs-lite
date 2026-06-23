package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver

	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3/contracttest"
)

// TestMultiDBContract verifies that the Postgres multi-DB split routes each
// concern to the correct database. Requires POSTGRES_TEST_DSN and the ability
// to CREATE DATABASE (typically a superuser or database owner).
func TestMultiDBContract(t *testing.T) {
	primaryDSN := os.Getenv("POSTGRES_TEST_DSN")
	if primaryDSN == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping Postgres multi-DB contract tests")
	}

	// Derive separate database DSNs by suffixing the database name.
	eventDSN := deriveDB(t, primaryDSN, "_events")
	queryDSN := deriveDB(t, primaryDSN, "_queries")
	viewDSN := deriveDB(t, primaryDSN, "_views")

	// Create the databases (must run before the preset opens them).
	createTestDB(t, primaryDSN, eventDSN)
	createTestDB(t, primaryDSN, queryDSN)
	createTestDB(t, primaryDSN, viewDSN)

	t.Cleanup(func() {
		dropTestDB(t, primaryDSN, eventDSN)
		dropTestDB(t, primaryDSN, queryDSN)
		dropTestDB(t, primaryDSN, viewDSN)
	})

	contracttest.RunMultiDBSuite(t, func(_ *testing.T) (*contracttest.MultiDBTest, error) {
		b, err := postgres.New(
			primaryDSN,
			postgres.WithEventDB(eventDSN),
			postgres.WithQueryDB(queryDSN),
			postgres.WithViewDB(viewDSN),
		)
		if err != nil {
			return nil, err
		}

		return &contracttest.MultiDBTest{
			Bundle:    b,
			EventDSN:  eventDSN,
			QueryDSN:  queryDSN,
			ViewDSN:   viewDSN,
			CountRows: countPostgresRows,
		}, nil
	})
}

// deriveDB replaces the database name in a Postgres DSN with name+suffix.
// Works with both URL format (postgres://host/db) and keyword format
// (host=... dbname=db).
func deriveDB(t *testing.T, dsn, suffix string) string {
	t.Helper()

	// URL format: postgres://user:pass@host:port/dbname?params
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// Find the last path segment before ?
		trimmed := dsn
		if idx := strings.Index(trimmed, "?"); idx >= 0 {
			trimmed = trimmed[:idx]
		}

		lastSlash := strings.LastIndex(trimmed, "/")
		if lastSlash >= 0 {
			return dsn[:lastSlash+1] + trimmed[lastSlash+1:] + suffix + dsn[len(trimmed):]
		}
	}

	// Keyword format: host=... dbname=db
	if idx := strings.Index(dsn, "dbname="); idx >= 0 {
		start := idx + len("dbname=")
		rest := dsn[start:]
		end := strings.IndexAny(rest, " ")
		if end < 0 {
			end = len(rest)
		}

		return dsn[:start] + rest[:end] + suffix + dsn[start+end:]
	}

	t.Fatalf("cannot parse Postgres DSN to derive database name: %s", dsn)

	return ""
}

// dbNameFromDSN extracts the database name from a DSN for CREATE/DROP.
func dbNameFromDSN(t *testing.T, dsn string) string {
	t.Helper()

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		trimmed := dsn
		if idx := strings.Index(trimmed, "?"); idx >= 0 {
			trimmed = trimmed[:idx]
		}

		lastSlash := strings.LastIndex(trimmed, "/")
		if lastSlash >= 0 {
			return trimmed[lastSlash+1:]
		}
	}

	if idx := strings.Index(dsn, "dbname="); idx >= 0 {
		rest := dsn[idx+len("dbname="):]
		end := strings.IndexAny(rest, " ")
		if end < 0 {
			return rest
		}

		return rest[:end]
	}

	t.Fatalf("cannot extract database name from DSN: %s", dsn)

	return ""
}

func createTestDB(t *testing.T, adminDSN, dbDSN string) {
	t.Helper()

	dbName := dbNameFromDSN(t, dbDSN)

	// Connect to the maintenance database to create the test database.
	sqlDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	_, err = sqlDB.ExecContext(context.Background(),
		fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create database %s: %v", dbName, err)
	}
}

func dropTestDB(t *testing.T, adminDSN, dbDSN string) {
	t.Helper()

	dbName := dbNameFromDSN(t, dbDSN)

	dropDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return
	}
	defer func() { _ = dropDB.Close() }()

	_, _ = dropDB.ExecContext(context.Background(),
		fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName))
}

func countPostgresRows(t *testing.T, dsn, table string) int {
	t.Helper()

	countDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}
	defer func() { _ = countDB.Close() }()

	var got int

	err = countDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&got)
	if err != nil {
		t.Fatalf("count %s.%s: %v", dbNameFromDSN(t, dsn), table, err)
	}

	return got
}
