package stack

// Capabilities describes what a [Bundle] can do — a machine-checkable
// tradeoff matrix. Each preset populates this struct at construction time so
// that consumers, benchmark tools, and documentation generators can
// introspect backend properties without trial-and-error.
//
// The fields are deliberately simple booleans and enums (not function
// pointers or interfaces) so that the struct is comparable, serializable,
// and usable in data-driven benchmarks.
type Capabilities struct {
	// Backend is the human-readable backend name, e.g. "sqlite", "pebble".
	Backend string

	// Persistent is true when data survives process restart. memory = false;
	// all disk-backed backends = true.
	Persistent bool

	// Distributed is true when the bus propagates events across processes.
	// postgres.WithDistributedBus = true; all others = false.
	Distributed bool

	// DurabilityRange lists the [DurabilityTier] values the backend supports.
	// Every backend supports at least DurabilityNormal.
	DurabilityRange []DurabilityTier

	// OLAP is true when the backend is optimised for analytical workloads
	// (columnar scans, GROUP BY). duckdb = true; all others = false.
	OLAP bool

	// CGoRequired is true when the backend requires CGo to compile.
	// duckdb = true; all others = false (pure-Go drivers).
	CGoRequired bool

	// Embedded is true when the database runs in-process (no external server).
	// memory, sqlite, pebble, turso, duckdb = true; postgres = false.
	Embedded bool

	// SyncEnabled is true when the backend supports remote synchronisation
	// (push/pull to a remote server). turso = true; all others = false.
	SyncEnabled bool
}

// WithCapabilities sets the [Capabilities] metadata on the Bundle. Presets
// call this during construction to declare their backend's properties.
// Consumers typically do not call this directly — the preset they choose
// determines the capabilities.
func WithCapabilities(caps Capabilities) Option {
	return func(b *Bundle) { b.capabilities = caps }
}

// Capabilities returns the [Capabilities] struct describing this Bundle's
// backend. Returns a zero-value Capabilities (Backend = "") when no preset
// set it — consumers should treat an empty Backend as "unknown".
func (b *Bundle) Capabilities() Capabilities {
	return b.capabilities
}
