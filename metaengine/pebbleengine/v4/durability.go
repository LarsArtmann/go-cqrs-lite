package pebbleengine

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// tierToOptions maps a durability tier to Pebble engine options — the
// translation table from the stack presets (proposal §5, 2026-08-17):
//
//	strict  → WAL enabled, sync writes (fsync per write — the engine default)
//	normal  → WAL enabled, async writes (no per-write fsync — safe against
//	          app crash, small loss window on kernel crash)
//	relaxed → DisableWAL + async writes (memtable only; loss on crash)
//
// The empty tier maps to no options: engine defaults apply.
func tierToOptions(tier metaengine.DurabilityTier) ([]Option, error) {
	if err := metaengine.ValidateDurabilityTier(tier); err != nil {
		return nil, fmt.Errorf("pebbleengine: %w", err)
	}

	switch tier {
	case metaengine.DurabilityStrict:
		return nil, nil
	case metaengine.DurabilityNormal:
		return []Option{WithAsyncWrites()}, nil
	case metaengine.DurabilityRelaxed:
		return []Option{WithAsyncWrites(), WithDisableWAL()}, nil
	default:
		return nil, nil
	}
}
