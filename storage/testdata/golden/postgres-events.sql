CREATE TABLE IF NOT EXISTS events (
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
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);
