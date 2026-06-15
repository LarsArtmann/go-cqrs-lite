package sql

const (
	// TableCommands is the SQL table name for commands.
	TableCommands = "commands"
	// TableEvents is the SQL table name for events.
	TableEvents = "events"
	// TableSnapshots is the SQL table name for snapshots.
	TableSnapshots = "snapshots"
	// TableCheckpoints is the SQL table name for checkpoints.
	TableCheckpoints = "checkpoints"
	// TableQueries is the SQL table name for queries.
	TableQueries = "queries"
)

// EventColumns is the standard SELECT column list for event queries.
const EventColumns = "id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at"

// CommandColumns is the standard SELECT column list for command queries.
const CommandColumns = "id, command_type, aggregate_type, aggregate_id, payload, metadata, received_at"

// QueryColumns is the standard SELECT column list for query queries.
const QueryColumns = "id, query_type, payload, metadata, received_at"
