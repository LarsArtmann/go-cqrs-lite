package view

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

func openSQLiteInMemory() (*sql.DB, error) {
	return sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
}

func mustOpenDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := openSQLiteInMemory()
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}

	return db
}
