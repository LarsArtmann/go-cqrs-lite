package pgengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// withDurabilityTier records the durability tier the driver factory applied
// to the DSN (synchronous_commit). Unexported: the tier is engine-internal
// state, not a consumer-facing option — consumers name the tier in
// DriverConfig.Durability and the DSN carries the behavior.
func withDurabilityTier(tier metaengine.DurabilityTier) Option {
	return func(e *pgEngine) { e.durability = tier }
}

// EffectiveDurability implements metaengine.DurabilityReporter. The effective
// tier is the one the driver factory translated into the connection DSN
// (synchronous_commit mapping in durability.go: strict → on, normal/relaxed
// → off). An engine constructed directly with New(dsn) reports
// engine-default (empty) — the server's own synchronous_commit setting is
// authoritative and unknowable client-side.
func (e *pgEngine) EffectiveDurability() metaengine.DurabilityTier {
	return e.durability
}
