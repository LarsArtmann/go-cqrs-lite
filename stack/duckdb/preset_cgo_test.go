//go:build cgo

package duckdb_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		return duckdb.New(filepath.Join(t.TempDir(), "contract.db"))
	})
}

func TestNew_InMemory(t *testing.T) {
	b, err := duckdb.New("")
	if err != nil {
		t.Fatalf("New(\"\"): %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventSink not set")
	}

	if b.ReadModels == nil {
		t.Fatal("ReadModels not set")
	}
}

func TestNew_PersistentFile(t *testing.T) {
	dir := t.TempDir()

	b, err := duckdb.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()
}

func TestNew_WithThreads(t *testing.T) {
	b, err := duckdb.New("", duckdb.WithThreads(2))
	if err != nil {
		t.Fatalf("New with threads: %v", err)
	}

	defer func() { _ = b.Close() }()
}

func TestNew_WithMemoryLimit(t *testing.T) {
	b, err := duckdb.New("", duckdb.WithMemoryLimit("512MB"))
	if err != nil {
		t.Fatalf("New with memory limit: %v", err)
	}

	defer func() { _ = b.Close() }()
}

func TestNew_WithoutAutoMigrate(t *testing.T) {
	b, err := duckdb.New(
		"",
		duckdb.WithDSN(sqlopt.WithoutAutoMigrate()),
	)
	if err != nil {
		t.Fatalf("New without auto-migrate: %v", err)
	}

	defer func() { _ = b.Close() }()
}

func TestNew_MultiDB(t *testing.T) {
	dir := t.TempDir()

	b, err := duckdb.New(
		filepath.Join(dir, "primary.db"),
		duckdb.WithDSN(
			sqlopt.WithEventDB(filepath.Join(dir, "events.db")),
			sqlopt.WithQueryDB(filepath.Join(dir, "queries.db")),
			sqlopt.WithViewDB(filepath.Join(dir, "views.db")),
		),
	)
	if err != nil {
		t.Fatalf("New multi-DB: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventStore not set")
	}
}

func TestMultiDBContract(t *testing.T) {
	contracttest.RunMultiDBSuite(t, func(t *testing.T) (*contracttest.MultiDBTest, error) {
		dir := t.TempDir()

		eventDSN := filepath.Join(dir, "events.db")
		queryDSN := filepath.Join(dir, "queries.db")
		viewDSN := filepath.Join(dir, "views.db")

		b, err := duckdb.New(
			filepath.Join(dir, "primary.db"),
			duckdb.WithDSN(
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
			CountRows: countDuckDBRows,
		}, nil
	})
}

func countDuckDBRows(t *testing.T, dsn, table string) int {
	t.Helper()

	sqlDB, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", filepath.Base(dsn), err)
	}

	defer func() { _ = sqlDB.Close() }()

	var count int

	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}

	return count
}

func TestBundle_CloseIsIdempotent(t *testing.T) {
	b, err := duckdb.New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_ = b.Close()
	_ = b.Close()
}

func TestBundle_HealthCheck(t *testing.T) {
	b, err := duckdb.New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if err := b.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}
