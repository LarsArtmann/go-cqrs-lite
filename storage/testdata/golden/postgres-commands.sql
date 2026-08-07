CREATE TABLE IF NOT EXISTS commands (
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
CREATE INDEX IF NOT EXISTS idx_commands_received_at ON commands(received_at);
