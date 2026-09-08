package sql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	return parseTimePointer(src, "postgres")
}

func (PostgresDialect) ExcludedRef(col string) string { return "excluded." + col }

func (PostgresDialect) OnConflictDoNothing(_ string) string {
	return "ON CONFLICT DO NOTHING" //nolint:goconst // SQL literal
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
    stream_type  VARCHAR(255) NOT NULL,
    stream_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           JSONB NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stream_type, stream_id)
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
