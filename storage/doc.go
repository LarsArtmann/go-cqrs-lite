// Package storage provides persistent event store implementations backed by
// SQL databases (PostgreSQL, SQLite, Turso) and Pebble (embedded KV).
package storage

import (
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

type Dialect = sqlpkg.Dialect
type PostgresDialect = sqlpkg.PostgresDialect
type SQLiteDialect = sqlpkg.SQLiteDialect
type OutboxStatus = sqlpkg.OutboxStatus

var (
	OutboxStatusPending      = sqlpkg.OutboxStatusPending
	OutboxStatusAcked        = sqlpkg.OutboxStatusAcked
	ErrNilDB                 = sqlpkg.ErrNilDB
	ErrUnexpectedTimeType    = sqlpkg.ErrUnexpectedTimeType
	ErrUnsupportedTimestamp  = sqlpkg.ErrUnsupportedTimestamp
	ErrConcurrencyConflict   = sqlpkg.ErrConcurrencyConflict
)

func Schema() string         { return sqlpkg.Schema() }
func SQLiteSchema() string { return sqlpkg.SQLiteSchema() }
