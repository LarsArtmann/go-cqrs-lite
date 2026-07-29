package storage

import (
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"
)

// OpenDuckDB opens a DuckDB database at the given path.
// Pass "" for an in-memory database.
//
// The caller is responsible for importing the DuckDB driver:
//
//	import _ "github.com/duckdb/duckdb-go/v2"
//
// (The driver requires CGo. See stack/duckdb for the CGo-enabled preset.)
func OpenDuckDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.open_duckdb",
			"open duckdb database at "+dbPath,
		)
	}

	return db, nil
}

// OpenDuckDBInMemory opens an in-memory DuckDB database.
//
// The caller is responsible for importing the DuckDB driver:
//
//	import _ "github.com/duckdb/duckdb-go/v2"
func OpenDuckDBInMemory() (*sql.DB, error) {
	return OpenDuckDB("")
}
