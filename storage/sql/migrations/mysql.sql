-- MySQL schema for go-cqrs-lite storage module.
-- Embedded via //go:embed in storage/sql/schema_embed.go.
-- MySQL uses indexes embedded in CREATE TABLE (no CREATE INDEX IF NOT EXISTS).

CREATE TABLE IF NOT EXISTS events (
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
);

CREATE TABLE IF NOT EXISTS commands (
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
);

CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    VARCHAR(255) NOT NULL,
    version         INTEGER NOT NULL,
    state           JSON NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (aggregate_type, aggregate_id)
);

CREATE TABLE IF NOT EXISTS queries (
    id               VARCHAR(255) PRIMARY KEY,
    query_type       VARCHAR(255) NOT NULL,
    payload          LONGBLOB,
    metadata         JSON,
    received_at      DATETIME(3) NOT NULL,
    created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_queries_type (query_type),
    KEY idx_queries_received_at (received_at)
);

CREATE TABLE IF NOT EXISTS checkpoints (
    projection_name VARCHAR(255) PRIMARY KEY,
    event_id        VARCHAR(255) NOT NULL,
    processed_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS cqrs_kv (
    `key`   VARBINARY(512) PRIMARY KEY,
    `value` LONGBLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS timers (
    id         VARCHAR(255) PRIMARY KEY,
    fire_at    DATETIME(3) NOT NULL,
    payload    LONGBLOB NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_timers_fire_at (fire_at)
);
