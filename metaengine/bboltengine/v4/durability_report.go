package bboltengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EffectiveDurability implements metaengine.DurabilityReporter. bbolt fsyncs
// every transaction commit unless WithNoSync was set: sync → strict (normal
// is an alias for the same settings — bbolt has no WAL, so normal can never
// be a durability drop), NoSync → relaxed. The in-memory/temp engine reports
// engine-default (empty) — nothing it writes is durable in any tier's sense.
func (e *bboltEngine) EffectiveDurability() metaengine.DurabilityTier {
	if e.persistence == metaengine.PersistenceVolatile {
		return ""
	}

	if e.noSync {
		return metaengine.DurabilityRelaxed
	}

	return metaengine.DurabilityStrict
}
