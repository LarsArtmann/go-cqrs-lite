-- DuckDB schema for go-cqrs-lite storage module.
-- Embedded via //go:embed in storage/sql/schema_embed.go.
-- DuckDB is PostgreSQL-compatible: VARCHAR, BLOB, TIMESTAMP WITH TIME ZONE,
-- CURRENT_TIMESTAMP, CREATE INDEX IF NOT EXISTS all work.

CREATE TABLE IF NOT EXISTS events (
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
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);

CREATE TABLE IF NOT EXISTS commands (
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
CREATE INDEX IF NOT EXISTS idx_commands_received_at ON commands(received_at);

CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR NOT NULL,
    aggregate_id    VARCHAR NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (aggregate_type, aggregate_id)
);

CREATE TABLE IF NOT EXISTS queries (
    id               VARCHAR PRIMARY KEY,
    query_type       VARCHAR NOT NULL,
    payload          BLOB,
    metadata         BLOB,
    received_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_queries_type ON queries(query_type);
CREATE INDEX IF NOT EXISTS idx_queries_received_at ON queries(received_at);

CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR PRIMARY KEY,
    event_id        VARCHAR NOT NULL,
    processed_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cqrs_kv (
    key   BLOB PRIMARY KEY,
    value BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS timers (
    id         VARCHAR PRIMARY KEY,
    fire_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    payload    BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);
