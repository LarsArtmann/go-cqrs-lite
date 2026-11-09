// Package cqrs is the CUE surface for system.DeploymentConfig (ADR-0165).
//
// CUE is an AUTHORING artifact, evaluated by the cue CLI at build/deploy
// time — the Go modules carry no CUE dependency. Export to the wire format
// system.LoadConfig reads:
//
//	cue eval -e standard --out yaml system/config.cue app.cue > deployment.yaml
//	cue eval -e standard --out json system/config.cue app.cue > deployment.json
//
// Two entry points:
//
//   - standard — the canonical three-tier topology preset (the CUE twin of
//     system.Standard): WAL + query audit log on the engine named "primary",
//     projections pooled over every declared engine. Add engines only:
//
//     standard: engines: primary: driver: "sqlite", dsn: "file:app.db"
//
//   - deployment — the schema alone, no instance defaults, for full control:
//
//     deployment: engines: primary: driver: "postgres", dsn: "postgres://..."
//     deployment: instances: [{role: "source-of-truth", engine: "primary"}]
package cqrs

// DeploymentConfig mirrors system.DeploymentConfig (koanf tags are the field
// names). Driver strings are OPEN: the metaengine registry can register more.
#DeploymentConfig: {
	// Engines maps named engine declarations (driver + DSN).
	engines: [string]: #EngineConfig

	// Buses maps named bus declarations ("gochannel" only today).
	buses?: [string]: #BusConfig

	// Instances bind roles to engines; leave unset when using standard.
	instances?: [...#InstanceConfig]

	// AcknowledgeWarnings lists scream-store warnings the operator ACKed
	// ("rule:target").
	acknowledge_warnings?: [...string]

	// ManifestPath pins the projection-plan manifest for drift detection.
	manifest_path?: string

	priority?: #PriorityConfig
}

#EngineConfig: {
	// driver: "sqlite", "memory", "pebble", "duckdb", "postgres", "turso", ...
	driver: string

	// dsn is the connection string (empty for memory).
	dsn?: string

	// pragmas are SQLite pragmas (e.g. "journal_mode=wal").
	pragmas?: [...string]

	priority?: #Priority

	// materialized_views declare operator-owned aggregate accelerations.
	materialized_views?: [...#MaterializedView]

	// max_open_conns raises the database/sql pool ceiling (0 = engine default).
	max_open_conns?: int & >=0
}

#MaterializedView: {
	collection: string

	// fn: "COUNT", "SUM", "MIN", "MAX", or "AVG".
	fn: "COUNT" | "SUM" | "MIN" | "MAX" | "AVG"

	// column is the aggregated JSON field (empty for COUNT).
	column?: string

	// group_by optionally groups by a JSON field (empty = scalar view).
	group_by?: string
}

#BusConfig: {
	// driver: "gochannel" (in-process; the only backend today).
	driver: string
	url?:   string

	// mode is introspection-only; publish is always synchronous. Removed at v5.
	mode?: string
}

#InstanceConfig: {
	role: "source-of-truth" | "events" | "commands" | "queries" | "snapshots" | "projections"

	// collections is introspection-only (surfaced in Introspection()).
	collections?: [...string]

	// engine names a single engine; engines lists a mixed pool
	// (projections). Mutually exclusive.
	engine?: string
	engines?: [...string]

	// durability: "strict" (fsync every commit), "normal" (crash-safe,
	// WAL-checkpoint window may drop on power loss), "relaxed" (rebuildable
	// tiers only). Unset = engine defaults.
	durability?: "strict" | "normal" | "relaxed"

	// publish lists bus names events fan out to (must be declared in buses).
	publish?: [...string]

	// subscribe is reserved, NOT read (may arrive at v5).
	subscribe?: [...string]

	cache?: #CacheConfig
}

#CacheConfig: {
	// engine is reserved and NOT read (removal at v5).
	engine?: string

	// capacity is the max entry count (otter W-TinyLFU evicts).
	capacity: int & >0
}

#Priority: "WriteSpeed" | "ReadSpeed" | "StorageSpace" | "Balanced"

#PriorityConfig: {
	global?: #Priority
	perEngine?: [string]: #Priority
	perQuery?: [string]:  #Priority
}

// deployment is the schema-only entry point for full manual control.
deployment: #DeploymentConfig

// standard is the canonical three-tier topology (ADR-0165):
//
//  1. source-of-truth on the engine named "primary" — events, commands,
//     queries, snapshots, checkpoints;
//  2. a dedicated queries audit instance on the same engine;
//  3. a projections instance pooling EVERY declared engine.
//
// Unlike the Go filler (which falls back to the alphabetically-first engine
// name), the CUE preset requires the conventional "primary" name for the WAL
// engine — CUE defaults cannot sort map keys.
standard: #DeploymentConfig & {
	instances: [
		{
			role:   "source-of-truth"
			engine: "primary"
			collections: ["events", "commands", "queries", "snapshots", "checkpoints"]
		},
		{
			role:   "queries"
			engine: "primary"
			collections: ["queries"]
		},
		{
			role: "projections"
			engines: [for name, _ in standard.engines {name}]
			collections: ["projections"]
		},
	]
}
