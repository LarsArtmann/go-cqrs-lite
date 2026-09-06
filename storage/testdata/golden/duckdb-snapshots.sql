CREATE TABLE IF NOT EXISTS snapshots (
    stream_type  VARCHAR NOT NULL,
    stream_id    VARCHAR NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stream_type, stream_id)
);
