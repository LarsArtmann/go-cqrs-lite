package stack

// DurabilityTier expresses the trade-off between write durability (data
// survives a crash) and write speed (how long each commit blocks).
//
// Each preset translates a DurabilityTier to its backend's native durability
// settings:
//
//   - SQLite:     PRAGMA synchronous (FULL / NORMAL / OFF)
//   - Turso:      PRAGMA synchronous (libSQL = SQLite fork)
//   - Postgres:   synchronous_commit (on / off)
//   - Pebble:     DisableWAL (true = Relaxed only)
//
// [DurabilityNormal] is the default — the same semantics every preset shipped
// before this type existed. Consumers who need strict durability on
// write-critical paths (financial ledgers, medical records) opt in explicitly
// with [WithDurability] or the preset-specific [WithDurability] adapter.
type DurabilityTier string

const (
	// DurabilityStrict maximises durability at the cost of write latency.
	// Every commit fsyncs to stable storage before returning.
	//
	// Backend translations:
	//
	//   - SQLite:   synchronous=FULL (fsync at every commit)
	//   - Postgres: synchronous_commit=on (WAL fsync per commit)
	//   - Pebble:   WAL enabled, no special async flags
	DurabilityStrict DurabilityTier = "strict"

	// DurabilityNormal balances durability and throughput. The default tier
	// for every preset — changing it would alter existing consumer semantics.
	//
	// Backend translations:
	//
	//   - SQLite:   synchronous=NORMAL (WAL mode: fsync only at checkpoint,
	//               not per commit — safe against app crash, not kernel crash)
	//   - Postgres: synchronous_commit=off (no per-commit WAL fsync — safe
	//               against app crash; small window of lost transactions on
	//               kernel crash)
	//   - Pebble:   WAL enabled, default flush behaviour
	DurabilityNormal DurabilityTier = "normal"

	// DurabilityRelaxed prioritises throughput. Data may be lost on process
	// or kernel crash. Suitable for caches, benchmarks, and ephemeral data.
	//
	// Backend translations:
	//
	//   - SQLite:   synchronous=OFF (no fsync ever — SQLite docs: "transaction
	//               will be lost if the application crashes")
	//   - Postgres: synchronous_commit=off + local synchronous_standby_names
	//   - Pebble:   DisableWAL=true (writes go to memtable only)
	DurabilityRelaxed DurabilityTier = "relaxed"
)

// WithDurability sets the durability tier on the Bundle. Each preset
// translates the tier to its backend's native durability settings during
// initialisation; this option records the chosen tier on the Bundle for
// introspection via [Bundle.Durability].
//
// Most consumers should use the preset-specific WithDurability adapter
// (sqlite.WithDurability, postgres.WithDurability, etc.) instead — those
// both apply the tier to the backend AND pass this option through to the
// Bundle.
//
// Default: [DurabilityNormal].
func WithDurability(tier DurabilityTier) Option {
	return func(b *Bundle) { b.durability = tier }
}

// Durability returns the durability tier the Bundle was constructed with.
// Default: [DurabilityNormal] (the value every preset used before this
// accessor existed). Consumers and benchmark tools use this to compare
// backends at the same durability level.
func (b *Bundle) Durability() DurabilityTier {
	if b.durability == "" {
		return DurabilityNormal
	}

	return b.durability
}
