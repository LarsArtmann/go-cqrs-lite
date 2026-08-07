CREATE TABLE IF NOT EXISTS snapshots (
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           JSONB NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_type, aggregate_id)
);
