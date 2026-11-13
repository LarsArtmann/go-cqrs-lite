package metaengine

import (
	"context"
	"fmt"
)

// Self-registration of the built-in memory engine. Like all other engines
// (sqlite, pebble, duckdb, etc.), the memory engine registers itself at init
// time so that operators can select it via config without any code changes.
func init() {
	RegisterDriver("memory", func(_ context.Context, cfg DriverConfig) (Engine, error) {
		if err := ValidateDurabilityTier(cfg.Durability); err != nil {
			return nil, fmt.Errorf("driver %q: %w", "memory", err)
		}

		// The memory engine holds no state on stable storage at all. It can
		// honor an explicit relaxed/normal tier as advisory (the operator was
		// warned by the scream rules), but a strict request — every commit
		// fsyncs — is a guarantee memory cannot make. Fail construction
		// rather than lie about durability.
		if cfg.Durability == DurabilityStrict {
			return nil, fmt.Errorf(
				"driver %q cannot provide %q durability — data lives only in process memory: %w",
				"memory", DurabilityStrict, ErrUnsupportedDurability,
			)
		}

		// ADR-0141 cell versioning: version chains are an in-memory
		// capability, enabled through the versioning constructor (the
		// retention policy trims the chain on write).
		if cfg.CellVersioning != nil {
			if err := cfg.CellVersioning.Validate(); err != nil {
				return nil, fmt.Errorf("driver %q cell_versioning: %w", "memory", err)
			}

			return NewMemoryEngineWithVersioning(WithRetention(*cfg.CellVersioning)), nil
		}

		return NewMemoryEngine(), nil
	})
}
