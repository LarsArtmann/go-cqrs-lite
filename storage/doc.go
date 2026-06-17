// Package storage provides persistent event store implementations backed by
// SQL databases (PostgreSQL, SQLite, Turso) and Pebble (embedded KV).
//
// # SQLite JSON1 Support
//
// Both modernc.org/sqlite (pure Go) and mattn/go-sqlite3 (CGo) ship with
// JSON1 enabled by default. Event metadata is stored as a JSON column, enabling
// in-database querying via json_extract:
//
//	SELECT * FROM events
//	WHERE json_extract(metadata, '$.correlation_id') = 'abc-123';
//
//	SELECT aggregate_type, COUNT(*) FROM events
//	WHERE json_extract(metadata, '$.user_role') = 'admin'
//	GROUP BY aggregate_type;
//
// For projection building, combine json_extract with json_each to iterate
// array-valued metadata fields directly in SQL without post-processing in Go.
package storage

import (
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

type (
	SQLiteDialect = sqlpkg.SQLiteDialect
)

var ErrNilDB = sqlpkg.ErrNilDB
