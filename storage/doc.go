// Package storage provides persistent event store implementations backed by
// SQL databases (PostgreSQL, SQLite, Turso) and Pebble (embedded KV).
package storage

import (
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

type (
	SQLiteDialect = sqlpkg.SQLiteDialect
)

var (
	ErrNilDB = sqlpkg.ErrNilDB
)
