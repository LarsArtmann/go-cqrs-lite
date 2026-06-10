package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func testQueryConfig() QueryConfig[string] {
	return QueryConfig[string]{
		Columns:    "id, name",
		Table:      "test_table",
		ScanRows:   testScanRows,
		WrapError:  func(err error, code, msg string) error { return fmt.Errorf("wrap: %w", err) },
		WrapEmpty:  func(err error, code, msg string) error { return event.WrapRejection(err, code, msg) },
		NotFound:   errors.New("not found"),
		DomainNoun: "items",
	}
}

func testScanRows(rows *sql.Rows) ([]string, error) {
	var results []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		results = append(results, s)
	}

	return results, nil
}

func TestQueryRows_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name FROM test_table").
		WithArgs("User", "123").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("alice"))

	results, err := QueryRows(context.Background(), db, SQLiteDialect{}, testQueryConfig(),
		LoadParams{Where: "", ErrMsg: "test"}, "User", "123")
	if err != nil {
		t.Fatalf("QueryRows: %v", err)
	}
	if len(results) != 1 || results[0] != "alice" {
		t.Errorf("results = %v, want [alice]", results)
	}
}

func TestQueryRows_QueryError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name FROM test_table").
		WithArgs("User", "123").
		WillReturnError(errors.New("connection lost"))

	cfg := testQueryConfig()
	_, err = QueryRows(context.Background(), db, SQLiteDialect{}, cfg,
		LoadParams{Where: "", ErrMsg: "test"}, "User", "123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQueryRows_RequireHitEmpty(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name FROM test_table").
		WithArgs("User", "123").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))

	cfg := testQueryConfig()
	_, err = QueryRows(context.Background(), db, SQLiteDialect{}, cfg,
		LoadParams{Where: "", RequireHit: true, ErrMsg: "test"}, "User", "123")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestQueryRows_RequireHitFalse_EmptyOK(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name FROM test_table").
		WithArgs("User", "123").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))

	cfg := testQueryConfig()
	results, err := QueryRows(context.Background(), db, SQLiteDialect{}, cfg,
		LoadParams{Where: "", RequireHit: false, ErrMsg: "test"}, "User", "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %v, want empty", results)
	}
}

func TestQueryRows_ExtraArgs(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, name FROM test_table").
		WithArgs("User", "123", 42).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("bob"))

	cfg := testQueryConfig()
	results, err := QueryRows(context.Background(), db, SQLiteDialect{}, cfg,
		LoadParams{
			Where:     "AND version > ?",
			ExtraArgs: []any{42},
			ErrMsg:    "test",
		}, "User", "123")
	if err != nil {
		t.Fatalf("QueryRows: %v", err)
	}
	if len(results) != 1 || results[0] != "bob" {
		t.Errorf("results = %v, want [bob]", results)
	}
}

func TestLoadWithSpan_CheckClosed(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	closedErr := errors.New("store is closed")
	_, err = LoadWithSpan(context.Background(), db, SQLiteDialect{},
		func() error { return closedErr },
		testQueryConfig(),
		LoadParams{SpanName: "test", CountAttr: "test.count"},
		"User", "123")
	if !errors.Is(err, closedErr) {
		t.Errorf("err = %v, want closedErr", err)
	}
}
