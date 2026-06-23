// Package storage provides persistent event store implementations backed by
// SQL databases (PostgreSQL, SQLite, Turso) and Pebble (embedded KV).
//
// # Schema Migration
//
// The raw constructors (NewSQLiteEventStore, NewSQLEventStore, etc.) do NOT
// auto-migrate. You must call SQLiteInitSchema or PostgresInitSchema before
// first use, or tables will be missing and operations will fail. The stack
// presets (stack/sqlite, stack/postgres, stack/turso) handle this automatically.
//
// If you are wiring stores manually (without a stack preset), add this after
// opening the database:
//
//	_ = storage.SQLiteInitSchema(ctx, db)   // SQLite
//	_ = storage.PostgresInitSchema(ctx, db) // Postgres
//
// Or better, use a stack preset which handles schema, WAL, busy_timeout, and
// lifecycle automatically:
//
//	bundle, _ := sqlite.New("app.db") // schema + WAL + stores + bus + Close
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
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

type (
	SQLiteDialect = sqlpkg.SQLiteDialect
)

var ErrNilDB = sqlpkg.ErrNilDB
