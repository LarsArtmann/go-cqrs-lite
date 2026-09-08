package sql

import (
	"strings"
	"time"
)

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
    stream_type  VARCHAR(255) NOT NULL,
    stream_id    VARCHAR(255) NOT NULL,
    version         INTEGER NOT NULL,
    state           JSON NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (stream_type, stream_id)
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
