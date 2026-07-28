//go:build cgo

// Package duckdb registers the DuckDB database/sql driver.
// This file is excluded when CGO_ENABLED=0 to keep the module buildable
// in pure-Go environments. Tests that need the actual DuckDB engine are
// guarded by the same build tag.
package duckdb

import _ "github.com/duckdb/duckdb-go/v2"
