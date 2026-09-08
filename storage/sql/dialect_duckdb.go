package sql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
    stream_type  VARCHAR NOT NULL,
    stream_id    VARCHAR NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stream_type, stream_id)
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
