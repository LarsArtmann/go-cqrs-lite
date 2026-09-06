CREATE TABLE IF NOT EXISTS snapshots (
    stream_type  TEXT NOT NULL,
    stream_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    state           BLOB NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (stream_type, stream_id)
);
