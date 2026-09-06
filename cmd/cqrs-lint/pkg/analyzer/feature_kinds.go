package analyzer

// StoreKind enumerates the persistence backends go-cqrs-lite supports.
type StoreKind string

const (
	StoreUnknown  StoreKind = "unknown"
	StoreSQLite   StoreKind = "sqlite"
	StorePostgres StoreKind = "postgres"
	StoreMySQL    StoreKind = "mysql"
	StorePebble   StoreKind = "pebble"
	StoreMemory   StoreKind = "memory"
	StoreTurso    StoreKind = "turso"
	StoreDuckDB   StoreKind = "duckdb"
	StoreBolt     StoreKind = "bolt"
	StoreCustom   StoreKind = "custom"
	StoreNone     StoreKind = "none"
)

// IsSQL reports whether this store kind is SQL-backed (capable of
// ORDER BY / WHERE pushdown). Used by adoption rules (F022) to gate
// pushdown-relevant suggestions. KV stores (Pebble, Bolt, Memory) are not
// SQL-backed — they cannot push down filters/sorts to the storage layer.
func (s StoreKind) IsSQL() bool {
	switch s {
	case StoreSQLite, StorePostgres, StoreMySQL, StoreDuckDB, StoreCustom:
		return true
	case StoreUnknown, StorePebble, StoreMemory, StoreTurso, StoreBolt, StoreNone:
		return false
	}

	return false
}

// IsEmbedded reports whether this store runs in-process (no separate server).
// Embedded stores (SQLite, Pebble, Bolt, Memory, DuckDB) share the application
// process. Distributed stores (Postgres, MySQL, Turso) run as a separate server.
func (s StoreKind) IsEmbedded() bool {
	switch s {
	case StoreSQLite, StorePebble, StoreBolt, StoreMemory, StoreDuckDB:
		return true
	case StoreUnknown, StorePostgres, StoreMySQL, StoreTurso, StoreCustom, StoreNone:
		return false
	}

	return false
}

// IsDistributed reports whether this store runs as a separate server process,
// enabling multi-instance deployment. Distributed stores require network I/O.
func (s StoreKind) IsDistributed() bool {
	switch s {
	case StorePostgres, StoreMySQL, StoreTurso:
		return true
	case StoreUnknown,
		StoreSQLite,
		StorePebble,
		StoreMemory,
		StoreDuckDB,
		StoreBolt,
		StoreCustom,
		StoreNone:
		return false
	}

	return false
}

// AllStoreKinds returns every defined StoreKind value, sorted alphabetically.
// Used by the explain command to derive valid config values programmatically
// instead of maintaining a hand-written copy.
func AllStoreKinds() []StoreKind {
	return []StoreKind{
		StoreSQLite, StorePostgres, StoreMySQL, StorePebble,
		StoreMemory, StoreTurso, StoreDuckDB, StoreBolt, StoreCustom, StoreNone,
	}
}

// CommandFlowKind classifies the command-dispatch pattern.
type CommandFlowKind string

const (
	CommandFlowUnknown  CommandFlowKind = "unknown"
	CommandFlowReadOnly CommandFlowKind = "read-only" // no dispatcher at all
	CommandFlowSync     CommandFlowKind = "sync"      // dispatcher present, batch/sync writes (no Dispatch)
	CommandFlowCommands CommandFlowKind = "commands"  // Dispatch() calls present
)

// AllCommandFlowKinds returns every defined CommandFlowKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllCommandFlowKinds() []CommandFlowKind {
	return []CommandFlowKind{
		CommandFlowReadOnly, CommandFlowSync, CommandFlowCommands,
	}
}

// AllTracingKinds returns every defined TracingKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllTracingKinds() []TracingKind {
	return []TracingKind{TracingOff, TracingOn}
}

// AllSnapshotKinds returns every defined SnapshotKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllSnapshotKinds() []SnapshotKind {
	return []SnapshotKind{SnapshotOff, SnapshotOn}
}

// AllDomainKinds returns every defined DomainKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllDomainKinds() []DomainKind {
	return []DomainKind{DomainFinancial, DomainInternal, DomainSecurity}
}

// MonetaryKind declares whether the project handles monetary values.
// Unknown defers to per-rule source heuristics; On/Off are explicit user
// declarations that override those heuristics.
type MonetaryKind string

const (
	MonetaryUnknown MonetaryKind = "unknown"
	MonetaryOn      MonetaryKind = "on"
	MonetaryOff     MonetaryKind = "off"
)

// AllMonetaryKinds returns every defined MonetaryKind value, sorted
// alphabetically. Excludes the Unknown sentinel.
func AllMonetaryKinds() []MonetaryKind {
	return []MonetaryKind{MonetaryOn, MonetaryOff}
}

// DomainKind classifies the business domain of a consumer project.
// Financial domains get stricter severity on security and money rules.
// The domain is auto-detected from event/command type names but can be
// overridden via config.
type DomainKind string

const (
	DomainUnknown   DomainKind = "unknown"
	DomainFinancial DomainKind = "financial"
	DomainInternal  DomainKind = "internal"
	DomainSecurity  DomainKind = "security"
)

// TracingKind indicates whether OTel tracing middleware is wired.
type TracingKind string

const (
	TracingUnknown TracingKind = "unknown"
	TracingOff     TracingKind = "off"
	TracingOn      TracingKind = "on"
)

// SnapshotKind indicates whether a snapshot store or strategy is configured.
type SnapshotKind string

const (
	SnapshotUnknown SnapshotKind = "unknown"
	SnapshotOff     SnapshotKind = "off"
	SnapshotOn      SnapshotKind = "on"
)

// String returns a human-readable multi-line summary for the doctor command
// and --verbose output.
