package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		return sqlite.New(filepath.Join(t.TempDir(), "contract.db"))
	})
}

func TestMultiDBContract(t *testing.T) {
	contracttest.RunMultiDBSuite(t, func(t *testing.T) (*contracttest.MultiDBTest, error) {
		dir := t.TempDir()

		eventDSN := filepath.Join(dir, "events.db")
		queryDSN := filepath.Join(dir, "queries.db")
		viewDSN := filepath.Join(dir, "views.db")

		b, err := sqlite.New(
			filepath.Join(dir, "primary.db"),
			sqlite.WithEventDB(eventDSN),
			sqlite.WithQueryDB(queryDSN),
			sqlite.WithViewDB(viewDSN),
		)
		if err != nil {
			return nil, err
		}

		return &contracttest.MultiDBTest{
			Bundle:    b,
			EventDSN:  eventDSN,
			QueryDSN:  queryDSN,
			ViewDSN:   viewDSN,
			CountRows: countSQLiteRows,
		}, nil
	})
}

func countSQLiteRows(t *testing.T, dsn, table string) int {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", filepath.Base(dsn), err)
	}
	defer func() { _ = db.Close() }()

	var got int

	err = db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&got)
	if err != nil {
		t.Fatalf("count %s.%s: %v", filepath.Base(dsn), table, err)
	}

	return got
}
