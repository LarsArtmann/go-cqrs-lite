package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// TestMultiDBContract verifies that the MySQL multi-DB split routes each
// concern to the correct database. Requires a MySQL instance (testcontainers).
func TestMultiDBContract(t *testing.T) {
	primaryDSN := mysqlDSN(t)

	eventDSN := deriveMySQLDB(t, primaryDSN, "_events")
	queryDSN := deriveMySQLDB(t, primaryDSN, "_queries")
	viewDSN := deriveMySQLDB(t, primaryDSN, "_views")

	createMySQLDB(t, primaryDSN, eventDSN)
	createMySQLDB(t, primaryDSN, queryDSN)
	createMySQLDB(t, primaryDSN, viewDSN)

	contracttest.RunMultiDBSuite(t, func(_ *testing.T) (*contracttest.MultiDBTest, error) {
		b, err := mysql.New(
			primaryDSN,
			mysql.WithDSN(
				sqlopt.WithEventDB(eventDSN),
				sqlopt.WithQueryDB(queryDSN),
				sqlopt.WithViewDB(viewDSN),
			),
		)
		if err != nil {
			return nil, err
		}

		return &contracttest.MultiDBTest{
			Bundle:    b,
			EventDSN:  eventDSN,
			QueryDSN:  queryDSN,
			ViewDSN:   viewDSN,
			CountRows: countMySQLRows,
		}, nil
	})
}

// deriveMySQLDB replaces the database name in a MySQL DSN with name+suffix.
// MySQL DSN format: user:pass@tcp(host:port)/dbname?params.
func deriveMySQLDB(t *testing.T, dsn, suffix string) string {
	t.Helper()

	slashIdx := strings.LastIndex(dsn, "/")
	if slashIdx < 0 {
		t.Fatalf("cannot parse MySQL DSN (no '/' found): %s", dsn)
	}

	queryIdx := strings.Index(dsn[slashIdx:], "?")
	if queryIdx >= 0 {
		base := dsn[:slashIdx+1]
		dbName := dsn[slashIdx+1 : slashIdx+queryIdx]
		params := dsn[slashIdx+queryIdx:]

		return base + dbName + suffix + params
	}

	return dsn + suffix
}

// mysqlDBName extracts the database name from a MySQL DSN.
func mysqlDBName(t *testing.T, dsn string) string {
	t.Helper()

	slashIdx := strings.LastIndex(dsn, "/")
	if slashIdx < 0 {
		t.Fatalf("cannot parse MySQL DSN: %s", dsn)
	}

	rest := dsn[slashIdx+1:]
	if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
		return rest[:qIdx]
	}

	return rest
}

func createMySQLDB(t *testing.T, adminDSN, dbDSN string) {
	t.Helper()

	dbName := mysqlDBName(t, dbDSN)

	sqlDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	_, err = sqlDB.ExecContext(context.Background(),
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName))
	if err != nil {
		t.Fatalf("create database %s: %v", dbName, err)
	}
}

func countMySQLRows(t *testing.T, dsn, table string) int {
	t.Helper()

	countDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}
	defer func() { _ = countDB.Close() }()

	var got int

	err = countDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM `"+table+"`").Scan(&got)
	if err != nil {
		t.Fatalf("count %s.%s: %v", mysqlDBName(t, dsn), table, err)
	}

	return got
}
