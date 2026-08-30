package sqliteengine

import (
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EffectiveDurability implements metaengine.DurabilityReporter. The tier is
// derived from the synchronous PRAGMA the engine was constructed with
// (journal_mode is always WAL): FULL → strict, NORMAL → normal, OFF →
// relaxed. No explicit synchronous pragma (the driver default applies)
// reports engine-default (empty) rather than guessing a tier.
func (e *sqliteEngine) EffectiveDurability() metaengine.DurabilityTier {
	return synchronousTier(e.synchronousPragma)
}

// synchronousTier maps a synchronous pragma value ("FULL"/"NORMAL"/"OFF") to
// its durability tier; empty input maps to engine-default.
func synchronousTier(value string) metaengine.DurabilityTier {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "FULL":
		return metaengine.DurabilityStrict
	case "NORMAL":
		return metaengine.DurabilityNormal
	case "OFF":
		return metaengine.DurabilityRelaxed
	default:
		return ""
	}
}
