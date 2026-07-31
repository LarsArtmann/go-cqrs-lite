package sql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Dialect abstracts SQL differences between database backends (PostgreSQL, SQLite,
// MySQL, DuckDB). Each store method delegates placeholder formatting, time handling,
// and upsert clause generation to a Dialect, eliminating the duplicated per-backend
// store pairs.
type Dialect interface { //nolint:interfacebloat // each method returns a distinct schema DDL
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
	TimerSchema() string

	// ExcludedRef returns the SQL reference to the "excluded" (would-be-inserted)
	// value of a column in an upsert SET clause.
	//
	// PostgreSQL/SQLite/DuckDB return "excluded.col".
	// MySQL returns "VALUES(col)".
	ExcludedRef(col string) string

	// OnConflictDoNothing generates the clause appended to an INSERT statement
	// for insert-or-ignore-on-conflict semantics (no update on conflict).
	//
	// noOpCol is a guaranteed-present column that MySQL uses for its
	// self-assignment no-op trick (ON DUPLICATE KEY UPDATE col = col).
	// PostgreSQL/SQLite/DuckDB ignore it and emit "ON CONFLICT DO NOTHING".
	OnConflictDoNothing(noOpCol string) string

	// OnConflictDoUpdate generates the clause appended to an INSERT statement
	// for insert-or-update-on-conflict semantics.
	//
	// conflictCols are the unique-constraint target columns (empty for MySQL,
	// which infers the conflict target from unique keys automatically).
	// setExprs are the SET assignments — build each with [Dialect.ExcludedRef]
	// (e.g. "version = excluded.version" or "version = VALUES(version)").
	//
	// PostgreSQL/SQLite/DuckDB: "ON CONFLICT(cols) DO UPDATE SET exprs".
	// MySQL: "ON DUPLICATE KEY UPDATE exprs".
	OnConflictDoUpdate(conflictCols []string, setExprs []string) string

	// QuoteIdentifier quotes a SQL identifier if the dialect requires it.
	// MySQL backtick-quotes reserved-word columns like `key`; all other dialects
	// return the name unchanged.
	QuoteIdentifier(name string) string
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

// parseTimePointer asserts src is a *time.Time, wraps the failure with a
// dialect-specific code/message, and returns the dereferenced value. Shared
// between Postgres and DuckDB (both native-time dialects); SQLite overrides
// ParseTime because it scans timestamp strings instead of *time.Time.
func parseTimePointer(src any, dialect string) (time.Time, error) {
	tp, ok := src.(*time.Time)
	if !ok {
		return time.Time{}, errorfamily.WrapCorruption(
			ErrUnexpectedTimeType,
			"storage.unexpected_time_type",
			fmt.Sprintf("%s dialect: expected *time.Time, got %T", dialect, src),
		)
	}

	return *tp, nil
}

func (PostgresDialect) ParseTime(src any) (time.Time, error) {
	return parseTimePointer(src, "postgres")
}

func (PostgresDialect) ExcludedRef(col string) string { return "excluded." + col }

func (PostgresDialect) OnConflictDoNothing(_ string) string {
	return "ON CONFLICT DO NOTHING"
}

func (PostgresDialect) OnConflictDoUpdate(conflictCols []string, setExprs []string) string {
	return fmt.Sprintf("ON CONFLICT(%s) DO UPDATE SET %s",
		strings.Join(conflictCols, ", "), strings.Join(setExprs, ", "))
}

func (PostgresDialect) QuoteIdentifier(name string) string { return name }

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

func (PostgresDialect) TimerSchema() string {
	return `CREATE TABLE IF NOT EXISTS timers (
    id         TEXT PRIMARY KEY,
    fire_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    payload    BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`
}

// MySQLDialect is the Dialect for MySQL and MariaDB databases.
// MySQL uses ? placeholders, native time.Time handling, and LONGBLOB/JSON types.
// Indexes are embedded in CREATE TABLE (MySQL lacks CREATE INDEX IF NOT EXISTS).
type MySQLDialect struct{}

func (MySQLDialect) Placeholder(_ int) string { return "?" }

func (MySQLDialect) FormatTime(t time.Time) any { return t }

func (MySQLDialect) ScanTimeDest() any {
	return new(time.Time)
}

func (MySQLDialect) ParseTime(src any) (time.Time, error) {
	return parseTimePointer(src, "mysql")
}

func (MySQLDialect) ExcludedRef(col string) string { return "VALUES(" + col + ")" }

func (MySQLDialect) OnConflictDoNothing(noOpCol string) string {
	return "ON DUPLICATE KEY UPDATE " + noOpCol + " = " + noOpCol
}

func (MySQLDialect) OnConflictDoUpdate(_ []string, setExprs []string) string {
	return "ON DUPLICATE KEY UPDATE " + strings.Join(setExprs, ", ")
}

func (MySQLDialect) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (MySQLDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id               VARCHAR(255) PRIMARY KEY,
    event_type       VARCHAR(255) NOT NULL,
    aggregate_type   VARCHAR(255) NOT NULL,
    aggregate_id     VARCHAR(255) NOT NULL,
    version          INTEGER NOT NULL,
    schema_version   INTEGER NOT NULL DEFAULT 1,
    payload          LONGBLOB,
    payload_encoding VARCHAR(32) NOT NULL DEFAULT 'json',
    metadata         JSON,
    occurred_at      DATETIME(3) NOT NULL,
    created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_events_agg_version (aggregate_type, aggregate_id, version),
    KEY idx_events_aggregate (aggregate_type, aggregate_id),
    KEY idx_events_type (event_type),
    KEY idx_events_occurred_at (occurred_at),
    KEY idx_events_agg_time (aggregate_type, aggregate_id, occurred_at)
);`
}

func (MySQLDialect) CommandSchema() string {
	return `CREATE TABLE IF NOT EXISTS commands (
    id               VARCHAR(255) PRIMARY KEY,
    command_type     VARCHAR(255) NOT NULL,
    aggregate_type   VARCHAR(255) NOT NULL,
    aggregate_id     VARCHAR(255) NOT NULL,
    payload          LONGBLOB,
    metadata         JSON,
    received_at      DATETIME(3) NOT NULL,
    created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_commands_aggregate (aggregate_type, aggregate_id),
    KEY idx_commands_type (command_type),
    KEY idx_commands_received_at (received_at)
);`
}

func (MySQLDialect) QuerySchema() string {
	return `CREATE TABLE IF NOT EXISTS queries (
    id               VARCHAR(255) PRIMARY KEY,
    query_type       VARCHAR(255) NOT NULL,
    payload          LONGBLOB,
    metadata         JSON,
    received_at      DATETIME(3) NOT NULL,
    created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_queries_type (query_type),
    KEY idx_queries_received_at (received_at)
);`
}

func (MySQLDialect) SnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    VARCHAR(255) NOT NULL,
    version         INTEGER NOT NULL,
    state           JSON NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

func (MySQLDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR(255) PRIMARY KEY,
    event_id        VARCHAR(255) NOT NULL,
    processed_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);`
}

func (MySQLDialect) KVSchema() string {
	return "CREATE TABLE IF NOT EXISTS cqrs_kv (\n" +
		"    `key`   VARBINARY(512) PRIMARY KEY,\n" +
		"    `value` LONGBLOB NOT NULL\n" +
		");"
}

func (MySQLDialect) TimerSchema() string {
	return `CREATE TABLE IF NOT EXISTS timers (
    id         VARCHAR(255) PRIMARY KEY,
    fire_at    DATETIME(3) NOT NULL,
    payload    LONGBLOB NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_timers_fire_at (fire_at)
);`
}

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

func (SQLiteDialect) TimerSchema() string {
	return `CREATE TABLE IF NOT EXISTS timers (
    id         TEXT PRIMARY KEY,
    fire_at    TEXT NOT NULL,
    payload    BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`
}

// DuckDBDialect is the Dialect for DuckDB databases.
//
// DuckDB is an embedded analytical (OLAP) SQL engine. Its SQL is largely
// PostgreSQL-compatible: numbered placeholders ($1, $2, …), TIMESTAMP WITH TIME
// ZONE, and CREATE INDEX IF NOT EXISTS all work as expected. The main
// differences from the Postgres dialect are the type names (BLOB instead of
// BYTEA, VARCHAR instead of TEXT) and the default-expression syntax
// (CURRENT_TIMESTAMP instead of NOW()).
//
// DuckDB returns time.Time values natively from its Go driver, so time handling
// mirrors the Postgres dialect.
type DuckDBDialect struct{}

func (DuckDBDialect) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

func (DuckDBDialect) FormatTime(t time.Time) any { return t }

func (DuckDBDialect) ScanTimeDest() any {
	return new(time.Time)
}

func (DuckDBDialect) ParseTime(src any) (time.Time, error) {
	return parseTimePointer(src, "duckdb")
}

func (DuckDBDialect) ExcludedRef(col string) string { return "excluded." + col }

func (DuckDBDialect) OnConflictDoNothing(_ string) string {
	return "ON CONFLICT DO NOTHING"
}

func (DuckDBDialect) OnConflictDoUpdate(conflictCols []string, setExprs []string) string {
	return fmt.Sprintf("ON CONFLICT(%s) DO UPDATE SET %s",
		strings.Join(conflictCols, ", "), strings.Join(setExprs, ", "))
}

func (DuckDBDialect) QuoteIdentifier(name string) string { return name }

func (DuckDBDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id               VARCHAR PRIMARY KEY,
    event_type       VARCHAR NOT NULL,
    aggregate_type   VARCHAR NOT NULL,
    aggregate_id     VARCHAR NOT NULL,
    version          INTEGER NOT NULL,
    schema_version   INTEGER NOT NULL DEFAULT 1,
    payload          BLOB,
    payload_encoding VARCHAR NOT NULL DEFAULT 'json',
    metadata         BLOB,
    occurred_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);`
}

func (DuckDBDialect) CommandSchema() string {
	return `CREATE TABLE IF NOT EXISTS commands (
    id               VARCHAR PRIMARY KEY,
    command_type     VARCHAR NOT NULL,
    aggregate_type   VARCHAR NOT NULL,
    aggregate_id     VARCHAR NOT NULL,
    payload          BLOB,
    metadata         BLOB,
    received_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_commands_aggregate ON commands(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_commands_type ON commands(command_type);
CREATE INDEX IF NOT EXISTS idx_commands_received_at ON commands(received_at);`
}

func (DuckDBDialect) SnapshotSchema() string {
	return `CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR NOT NULL,
    aggregate_id    VARCHAR NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (aggregate_type, aggregate_id)
);`
}

func (DuckDBDialect) QuerySchema() string {
	return `CREATE TABLE IF NOT EXISTS queries (
    id               VARCHAR PRIMARY KEY,
    query_type       VARCHAR NOT NULL,
    payload          BLOB,
    metadata         BLOB,
    received_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_queries_type ON queries(query_type);
CREATE INDEX IF NOT EXISTS idx_queries_received_at ON queries(received_at);`
}

func (DuckDBDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR PRIMARY KEY,
    event_id        VARCHAR NOT NULL,
    processed_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
}

func (DuckDBDialect) KVSchema() string {
	return `CREATE TABLE IF NOT EXISTS cqrs_kv (
    key   BLOB PRIMARY KEY,
    value BLOB NOT NULL
);`
}

func (DuckDBDialect) TimerSchema() string {
	return `CREATE TABLE IF NOT EXISTS timers (
    id         VARCHAR PRIMARY KEY,
    fire_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    payload    BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`
}

// Placeholders returns a comma-separated list of placeholders for the given count.
func Placeholders(d Dialect, count, offset int) string {
	parts := make([]string, count)

	for i := range count {
		parts[i] = d.Placeholder(offset + i + 1)
	}

	return strings.Join(parts, ", ")
}
