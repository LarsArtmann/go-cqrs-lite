CREATE TABLE IF NOT EXISTS snapshots (
    stream_type  VARCHAR(255) NOT NULL,
    stream_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           JSONB NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stream_type, stream_id)
);
