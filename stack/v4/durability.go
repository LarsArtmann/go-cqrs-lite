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
//   - Pebble:     WAL + per-write sync settings
//   - bbolt:      NoSync (bbolt has no WAL — see the exception notes below)
//
// [DurabilityNormal] is the default — the same semantics every preset shipped
// before this type existed. Consumers who need strict durability on
// write-critical paths (financial ledgers, medical records) opt in explicitly
// with [WithDurability] or the preset-specific [WithDurability] adapter.
// art-dupl:accept intentional cross-module duplicate — separate go.mod, values MUST match
type DurabilityTier string

const (
	// DurabilityStrict maximises durability at the cost of write latency.
	// Every commit fsyncs to stable storage before returning.
	//
	// Backend translations:
	//
	//   - SQLite:   synchronous=FULL (fsync at every commit)
	//   - Postgres: synchronous_commit=on (WAL fsync per commit)
	//   - Pebble:   WAL enabled, sync writes (fsync per write)
	//   - bbolt:    fsync on every commit (the bbolt default)
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
	//   - Pebble:   WAL enabled, async writes (no per-write fsync — safe
	//               against app crash; small window of lost writes on kernel
	//               crash)
	//   - bbolt:    same as Strict. EXCEPTION: bbolt has no WAL, so its only
	//               async knob (NoSync) skips the commit fsync entirely —
	//               a weaker guarantee than every other backend's Normal,
	//               and one bbolt upstream documents as dangerous. bbolt
	//               therefore cannot offer an app-crash-safe middle tier;
	//               the bbolt preset also defaults to Strict (not Normal)
	//               so this no-op alias can never silently become a
	//               durability drop.
	DurabilityNormal DurabilityTier = "normal"

	// DurabilityRelaxed prioritises throughput. Data may be lost on process
	// or kernel crash. Suitable for caches, benchmarks, and ephemeral data.
	//
	// Backend translations:
	//
	//   - SQLite:   synchronous=OFF (no fsync ever — SQLite docs: "transaction
	//               will be lost if the application crashes")
	//   - Postgres: synchronous_commit=off + local synchronous_standby_names
	//   - Pebble:   DisableWAL=true + async writes (writes go to memtable
	//               only; async matters — with the WAL disabled, a sync write
	//               forces a memtable flush, the slowest path pebble has)
	//   - bbolt:    NoSync + NoFreelistSync (skips the commit fsync — bbolt
	//               upstream: dangerous, data loss possible on crash)
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
