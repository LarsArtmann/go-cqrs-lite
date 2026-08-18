package bboltengine

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// tierToOptions maps a durability tier to bbolt engine options:
//
//	strict  → default sync-on-commit (fsync per transaction)
//	normal  → SAME as strict. bbolt has no WAL, so its only async knob
//	          (NoSync) skips the commit fsync entirely — a weaker guarantee
//	          than every other backend's normal tier, and one bbolt upstream
//	          documents as dangerous. Normal is accepted as an alias so
//	          deployments can name the tier without per-engine exceptions,
//	          never silently becoming a durability drop (see the exception
//	          note on stack.DurabilityNormal).
//	relaxed → WithNoSync (NoSync + NoFreelistSync — skips the commit fsync;
//	          data loss and possible corruption on unclean shutdown)
//
// The empty tier maps to no options: engine defaults apply.
func tierToOptions(tier metaengine.DurabilityTier) ([]Option, error) {
	if err := metaengine.ValidateDurabilityTier(tier); err != nil {
		return nil, fmt.Errorf("bboltengine: %w", err)
	}

	switch tier {
	case metaengine.DurabilityRelaxed:
		return []Option{WithNoSync()}, nil
	case metaengine.DurabilityStrict, metaengine.DurabilityNormal:
		return nil, nil
	default:
		return nil, nil
	}
}
