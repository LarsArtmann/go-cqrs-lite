package view

import (
	"database/sql"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

func openSQLiteInMemory() (*sql.DB, error) {
	return sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
}
