package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
	SnapshotSchema() string
	CheckpointSchema() string
	OutboxSchema() string
	SagaSchema() string
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
		return time.Time{}, fmt.Errorf(
			"postgres dialect: expected *time.Time, got %T: %w",
			src, ErrUnexpectedTimeType,
		)
	}

	return *tp, nil
}

func (PostgresDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    event_type      VARCHAR(255) NOT NULL,
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    payload         BYTEA,
    metadata        JSONB,
    occurred_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);`
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

func (PostgresDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR(255) PRIMARY KEY,
    event_id        TEXT NOT NULL
);`
}

func (PostgresDialect) SagaSchema() string {
	return `CREATE TABLE IF NOT EXISTS sagas (
    id           TEXT PRIMARY KEY,
    saga_type    VARCHAR(255) NOT NULL,
    status       VARCHAR(50) NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0,
    err_msg      TEXT,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sagas_status ON sagas(status);
CREATE INDEX IF NOT EXISTS idx_sagas_type ON sagas(saga_type);`
}

func (PostgresDialect) OutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT '` + string(OutboxStatusPending) + `'` + `,
    events      JSONB NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = '` + string(OutboxStatusPending) + `'` + `;`
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
		return time.Time{}, fmt.Errorf(
			"sqlite dialect: expected *string, got %T: %w",
			src, ErrUnexpectedTimeType,
		)
	}

	return parseSQLiteTimestamp(*sp)
}

func (SQLiteDialect) EventSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    payload         BLOB,
    metadata        TEXT,
    occurred_at     TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);`
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

func (SQLiteDialect) CheckpointSchema() string {
	return `CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name TEXT PRIMARY KEY,
    event_id        TEXT NOT NULL
);`
}

func (SQLiteDialect) SagaSchema() string {
	return `CREATE TABLE IF NOT EXISTS sagas (
    id           TEXT PRIMARY KEY,
    saga_type    TEXT NOT NULL,
    status       TEXT NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0,
    err_msg      TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sagas_status ON sagas(status);
CREATE INDEX IF NOT EXISTS idx_sagas_type ON sagas(saga_type);`
}

func (SQLiteDialect) OutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT '` + string(OutboxStatusPending) + `'` + `,
    events      TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = '` + string(OutboxStatusPending) + `'` + `;`
}

// placeholders returns a comma-separated list of placeholders for the given count.
func placeholders(d Dialect, count, offset int) string {
	parts := make([]string, count)

	for i := range count {
		parts[i] = d.Placeholder(offset + i + 1)
	}

	return strings.Join(parts, ", ")
}
