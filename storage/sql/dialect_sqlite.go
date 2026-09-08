package sql

import (
	"fmt"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// SQLiteDialect is the Dialect for SQLite databases.
type SQLiteDialect struct{}

func (SQLiteDialect) Placeholder(_ int) string { return "?" }

// sqliteTimeFormat is a fixed-width RFC3339 variant that always emits 9
// fractional digits. This guarantees correct lexicographic ordering on TEXT
// columns — time.RFC3339Nano trims trailing zeros, which breaks string
// comparison when one timestamp's fractional digits are a prefix of another's.
const sqliteTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

func (SQLiteDialect) FormatTime(t time.Time) any {
	return t.Format(sqliteTimeFormat)
}

func (SQLiteDialect) ScanTimeDest() any {
	return new(string)
}

func (SQLiteDialect) ParseTime(src any) (time.Time, error) {
	sp, ok := src.(*string)
	if !ok {
		return time.Time{}, errorfamily.WrapCorruption(
			ErrUnexpectedTimeType,
			"storage.unexpected_time_type",
			fmt.Sprintf("sqlite dialect: expected *string, got %T", src),
		)
	}

	return ParseSQLiteTimestamp(*sp)
}

func (SQLiteDialect) ExcludedRef(col string) string { return "excluded." + col }

func (SQLiteDialect) OnConflictDoNothing(_ string) string {
	return "ON CONFLICT DO NOTHING"
}

func (SQLiteDialect) OnConflictDoUpdate(conflictCols []string, setExprs []string) string {
	return fmt.Sprintf("ON CONFLICT(%s) DO UPDATE SET %s",
		strings.Join(conflictCols, ", "), strings.Join(setExprs, ", "))
}

func (SQLiteDialect) QuoteIdentifier(name string) string { return name }

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
    stream_type  TEXT NOT NULL,
    stream_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (stream_type, stream_id)
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

func (SQLiteDialect) TimerSchema() string {
	return `CREATE TABLE IF NOT EXISTS timers (
    id         TEXT PRIMARY KEY,
    fire_at    TEXT NOT NULL,
    payload    BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`
}
