package sql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Dialect abstracts SQL differences between database backends (PostgreSQL, SQLite).
// Each store method delegates placeholder formatting and time handling to a Dialect,
// eliminating the duplicated PostgreSQL/SQLite store pairs.
type Dialect interface {
	Placeholder(index int) string
	FormatTime(t time.Time) any
	ScanTimeDest() any
	ParseTime(src any) (time.Time, error)
	EventSchema() string
	CommandSchema() string
	QuerySchema() string
	SnapshotSchema() string
	CheckpointSchema() string
	KVSchema() string
}

// PostgresDialect is the Dialect for PostgreSQL databases.
type PostgresDialect struct{}

func (PostgresDialect) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

func (PostgresDialect) FormatTime(t time.Time) any { return t }

func (PostgresDialect) ScanTimeDest() any {
	return new(time.Time)
}

func (PostgresDialect) ParseTime(src any) (time.Time, error) {
	tp, ok := src.(*time.Time)
	if !ok {
		return time.Time{}, event.WrapCorruption(
			ErrUnexpectedTimeType,
			"storage.unexpected_time_type",
			fmt.Sprintf("postgres dialect: expected *time.Time, got %T", src),
		)
	}

	return *tp, nil
}

func (PostgresDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id               TEXT PRIMARY KEY,
    event_type       VARCHAR(255) NOT NULL,
    aggregate_type   VARCHAR(255) NOT NULL,
    aggregate_id     TEXT NOT NULL,
    version          INTEGER NOT NULL,
    schema_version   INTEGER NOT NULL DEFAULT 1,
    payload          BYTEA,
    payload_encoding TEXT NOT NULL DEFAULT 'json',
    metadata         JSONB,
    occurred_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);`
}

func (PostgresDialect) CommandSchema() string {
	return `CREATE TABLE IF NOT EXISTS commands (
    id               TEXT PRIMARY KEY,
    command_type     VARCHAR(255) NOT NULL,
    aggregate_type   VARCHAR(255) NOT NULL,
    aggregate_id     TEXT NOT NULL,
    payload          BYTEA,
    metadata         JSONB,
    received_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_commands_aggregate ON commands(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_commands_type ON commands(command_type);
CREATE INDEX IF NOT EXISTS idx_commands_received_at ON commands(received_at);`
}

func (PostgresDialect) SnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           JSONB NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

func (PostgresDialect) QuerySchema() string {
	return `CREATE TABLE IF NOT EXISTS queries (
    id               TEXT PRIMARY KEY,
    query_type       VARCHAR(255) NOT NULL,
    payload          BYTEA,
    metadata         JSONB,
    received_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_queries_type ON queries(query_type);
CREATE INDEX IF NOT EXISTS idx_queries_received_at ON queries(received_at);`
}

func (PostgresDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR(255) PRIMARY KEY,
    event_id        TEXT NOT NULL,
    processed_at    TIMESTAMP NOT NULL DEFAULT NOW()
);`
}

func (PostgresDialect) KVSchema() string {
	return `CREATE TABLE IF NOT EXISTS cqrs_kv (
    key   BYTEA PRIMARY KEY,
    value BYTEA NOT NULL
);`
}

// SQLiteDialect is the Dialect for SQLite databases.
type SQLiteDialect struct{}

func (SQLiteDialect) Placeholder(_ int) string { return "?" }

func (SQLiteDialect) FormatTime(t time.Time) any {
	return t.Format(time.RFC3339Nano)
}

func (SQLiteDialect) ScanTimeDest() any {
	return new(string)
}

func (SQLiteDialect) ParseTime(src any) (time.Time, error) {
	sp, ok := src.(*string)
	if !ok {
		return time.Time{}, event.WrapCorruption(
			ErrUnexpectedTimeType,
			"storage.unexpected_time_type",
			fmt.Sprintf("sqlite dialect: expected *string, got %T", src),
		)
	}

	return ParseSQLiteTimestamp(*sp)
}

func (SQLiteDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id               TEXT PRIMARY KEY,
    event_type       TEXT NOT NULL,
    aggregate_type   TEXT NOT NULL,
    aggregate_id     TEXT NOT NULL,
    version          INTEGER NOT NULL,
    schema_version   INTEGER NOT NULL DEFAULT 1,
    payload          BLOB,
    payload_encoding TEXT NOT NULL DEFAULT 'json',
    metadata         TEXT,
    occurred_at      TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);`
}

func (SQLiteDialect) CommandSchema() string {
	return `CREATE TABLE IF NOT EXISTS commands (
    id               TEXT PRIMARY KEY,
    command_type     TEXT NOT NULL,
    aggregate_type   TEXT NOT NULL,
    aggregate_id     TEXT NOT NULL,
    payload          BLOB,
    metadata         TEXT,
    received_at      TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_commands_aggregate ON commands(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_commands_type ON commands(command_type);
CREATE INDEX IF NOT EXISTS idx_commands_received_at ON commands(received_at);`
}

func (SQLiteDialect) SnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

func (SQLiteDialect) QuerySchema() string {
	return `CREATE TABLE IF NOT EXISTS queries (
    id               TEXT PRIMARY KEY,
    query_type       TEXT NOT NULL,
    payload          BLOB,
    metadata         TEXT,
    received_at      TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_queries_type ON queries(query_type);
CREATE INDEX IF NOT EXISTS idx_queries_received_at ON queries(received_at);`
}

func (SQLiteDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name TEXT PRIMARY KEY,
    event_id        TEXT NOT NULL,
    processed_at    TEXT NOT NULL DEFAULT(datetime('now'))
);`
}

func (SQLiteDialect) KVSchema() string {
	return `CREATE TABLE IF NOT EXISTS cqrs_kv (
    key   BLOB PRIMARY KEY,
    value BLOB NOT NULL
);`
}

// Placeholders returns a comma-separated list of placeholders for the given count.
func Placeholders(d Dialect, count, offset int) string {
	parts := make([]string, count)

	for i := range count {
		parts[i] = d.Placeholder(offset + i + 1)
	}

	return strings.Join(parts, ", ")
}
