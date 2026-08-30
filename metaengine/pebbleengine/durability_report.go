package pebbleengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EffectiveDurability implements metaengine.DurabilityReporter. The engine
// fsyncs every write (pebble.Sync) unless WithAsyncWrites was set: sync →
// strict, async → normal. Relaxed resolves to the same engine settings as
// normal for Pebble, so it reports as normal. The in-memory engine reports
// engine-default (empty) — nothing it writes is durable in any tier's sense.
func (e *pebbleEngine) EffectiveDurability() metaengine.DurabilityTier {
	if e.persistence == metaengine.PersistenceVolatile {
		return ""
	}

	if e.syncWrites {
		return metaengine.DurabilityStrict
	}

	return metaengine.DurabilityNormal
}
