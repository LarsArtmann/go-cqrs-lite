package metaengine

// Persistence declares whether an engine's data survives process exit.
// Modeled on DDIA Chapter 1 (Reliability: survivability axis).
//
// This is orthogonal to Replication (DDIA Ch5, topology), NetworkRTT
// (DDIA Ch1, distance), and stack.Durability (fsync tiers). Persistence
// answers a single binary question: "if the process exits, is the data gone?"
//
// The zero value is PersistenceVolatile — the safe default. If you forget
// to declare persistence, the planner assumes the worst and emits a WARN
// diagnostic, rather than silently assuming durability. This mirrors the
// Replication pattern (ReplicationNone = "" is the safe zero value).
//
// Three engines (SQLite, Pebble, DuckDB) are volatile OR persistent depending
// on constructor arguments (:memory: vs file path). Their Profile() sets the
// field dynamically at construction time.
type Persistence string

const (
	// PersistenceVolatile means data lives in process RAM and is lost on exit.
	// This is the zero value — the safe default that causes the planner to warn
	// loudly rather than silently assume durability.
	// Examples: Memory engine, SQLite ":memory:", Pebble vfs.NewMem(), DuckDB ":memory:".
	PersistenceVolatile Persistence = ""

	// PersistencePersistent means data survives process exit via a disk file
	// or a remote server. The projection can be reused after a restart without
	// rebuilding from the event log.
	// Examples: SQLite file, Pebble on disk, DuckDB file, Postgres.
	PersistencePersistent Persistence = "persistent"
)

// IsVolatile returns true if the engine's data is lost on process exit.
// Convenience for planner filtering and diagnostics.
func (p EngineProfile) IsVolatile() bool {
	return p.Persistence != PersistencePersistent
}

// IsPersistent returns true if the engine's data survives process exit.
// Convenience for planner filtering and diagnostics.
func (p EngineProfile) IsPersistent() bool {
	return p.Persistence == PersistencePersistent
}
