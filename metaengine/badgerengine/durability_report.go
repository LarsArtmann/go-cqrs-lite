package badgerengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EffectiveDurability implements metaengine.DurabilityReporter. The engine
// fsyncs every write unless WithAsyncWrites was set (or the tier option asked
// for non-strict): sync → strict, async → normal. Relaxed resolves to the
// same engine settings as normal for Badger, so it reports as normal. The
// in-memory engine reports engine-default (empty) — nothing it writes is
// durable in any tier's sense.
func (e *badgerEngine) EffectiveDurability() metaengine.DurabilityTier {
	if e.inMemory {
		return ""
	}

	if e.syncWrites {
		return metaengine.DurabilityStrict
	}

	return metaengine.DurabilityNormal
}
