package sql

const (
	// TableEvents is the SQL table name for events.
	TableEvents = "events"
	// TableSnapshots is the SQL table name for snapshots.
	TableSnapshots = "snapshots"
	// TableCheckpoints is the SQL table name for checkpoints.
	TableCheckpoints = "checkpoints"
)

// EventColumns is the standard SELECT column list for event queries.
const EventColumns = "id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at"
