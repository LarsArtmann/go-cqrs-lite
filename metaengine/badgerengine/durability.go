package badgerengine

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// tierToOptions maps a durability tier to Badger engine options:
//
//	strict  → sync writes (fsync per write — the engine default)
//	normal  → async writes (no per-write fsync — safe against app crash,
//	          small loss window on kernel crash)
//	relaxed → async writes. Badger's floor: there is no knob below
//	          SyncWrites=false — the value log is always written and replayed
//	          on open, so the engine stays app-crash safe. Relaxed names the
//	          least-durable settings badger offers rather than a lower tier.
//
// The empty tier maps to no options: engine defaults apply.
func tierToOptions(tier metaengine.DurabilityTier) ([]Option, error) {
	if err := metaengine.ValidateDurabilityTier(tier); err != nil {
		return nil, fmt.Errorf("badgerengine: %w", err)
	}

	switch tier {
	case metaengine.DurabilityStrict:
		return nil, nil
	case metaengine.DurabilityNormal, metaengine.DurabilityRelaxed:
		return []Option{WithAsyncWrites()}, nil
	default:
		return nil, nil
	}
}
