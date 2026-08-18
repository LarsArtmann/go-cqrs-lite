package metaengine

import (
	"fmt"
)

// DurabilityTier expresses the trade-off between write durability (data
// survives a crash) and write speed (how long each commit blocks). The tier
// travels on [DriverConfig] from the operator to the engine; each engine
// maps it to its native durability settings (SQLite: PRAGMA synchronous,
// Postgres: synchronous_commit, LSM stores: write-sync options).
//
// The zero value ("") means UNSPECIFIED: the engine applies its own defaults
// and no durability settings. This is what every deployment that does not
// name a tier gets, so it preserves pre-tier behavior exactly.
//
// Engines that do not implement per-tier behavior must fail construction on
// any non-empty tier via [RejectDurabilityTier] rather than silently
// ignoring the request — a silently dropped durability tier is a durability
// lie.
type DurabilityTier string

const (
	// DurabilityStrict maximises durability at the cost of write latency:
	// every commit fsyncs to stable storage before returning. Suitable for
	// write-critical paths (financial ledgers, medical records).
	DurabilityStrict DurabilityTier = "strict"

	// DurabilityNormal balances durability and throughput: safe against
	// application crash, with at most a small window of loss on kernel/power
	// failure. The tier most deployments should name explicitly.
	DurabilityNormal DurabilityTier = "normal"

	// DurabilityRelaxed prioritises throughput; data may be lost on process
	// or kernel crash. Suitable for caches, benchmarks, and rebuildable data.
	DurabilityRelaxed DurabilityTier = "relaxed"
)

// ValidateDurabilityTier reports whether tier is a valid DurabilityTier.
// The empty string is valid: unspecified, meaning engine defaults.
func ValidateDurabilityTier(tier DurabilityTier) error {
	switch tier {
	case "", DurabilityStrict, DurabilityNormal, DurabilityRelaxed:
		return nil
	default:
		return fmt.Errorf(
			"%w: %q (want %q, %q, or %q)",
			ErrUnsupportedDurability, string(tier),
			DurabilityStrict, DurabilityNormal, DurabilityRelaxed,
		)
	}
}

// RejectDurabilityTier fails construction when cfg requests a durability
// tier. Engines without per-tier behavior call it first in their driver
// factory so an explicit operator request is rejected loudly instead of
// silently ignored:
//
//	func init() {
//		metaengine.RegisterDriver("pebble", func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
//			if err := metaengine.RejectDurabilityTier("pebble", cfg); err != nil {
//				return nil, err
//			}
//			return NewPebbleEngine(cfg.DSN)
//		})
//	}
//
// An empty cfg.Durability passes: unspecified means engine defaults, which
// every engine supports.
func RejectDurabilityTier(driver string, cfg DriverConfig) error {
	if cfg.Durability == "" {
		return nil
	}

	if err := ValidateDurabilityTier(cfg.Durability); err != nil {
		return fmt.Errorf("driver %q: %w", driver, err)
	}

	return fmt.Errorf(
		"driver %q does not implement durability tiers (remove the durability setting or pick an engine that does): %w",
		driver,
		ErrUnsupportedDurability,
	)
}
