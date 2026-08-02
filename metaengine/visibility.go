package metaengine

import "time"

// VisibilityModel declares whether an engine's data is visible only to the
// local process or to all processes (eventually, via replication).
//
// All CQRS read models are eventually consistent — the projection host has
// measurable lag between an event being appended and the projection folding
// it. The visibility dimension captures whether that eventual consistency is
// bounded to a single process (VisibilityLocal) or spans all processes
// (VisibilityGlobal).
//
// See docs/planning/meta-engine-eventual-consistency-and-iroh.md for the
// full rationale.
type VisibilityModel string

const (
	// VisibilityLocal means the engine only sees writes from this process.
	// All current engines (Memory, SQLite, Pebble, DuckDB, Postgres) are local.
	// Local engines have near-zero replication lag (microseconds to milliseconds).
	VisibilityLocal VisibilityModel = "local"

	// VisibilityGlobal means the engine sees writes from all processes,
	// converging eventually via CRDT or similar replication.
	// Global engines (e.g. Iroh iroh-docs) have higher lag (milliseconds to seconds)
	// but enable cross-node data sharing without a central broker.
	VisibilityGlobal VisibilityModel = "global"
)

// DefaultTypicalLag is the assumed lag for local engines that don't specify
// one. It represents typical in-process projection processing latency.
const DefaultTypicalLag = 1 * time.Millisecond

// EffectiveVisibility returns the visibility model, defaulting to VisibilityLocal
// when the field is unset (zero value). This preserves backward compatibility —
// all existing engines are local.
func (p EngineProfile) EffectiveVisibility() VisibilityModel {
	if p.Visibility == "" {
		return VisibilityLocal
	}

	return p.Visibility
}

// EffectiveTypicalLag returns the typical lag, defaulting to DefaultTypicalLag
// when the field is unset. Used by the cost estimator as an additive latency
// component.
func (p EngineProfile) EffectiveTypicalLag() time.Duration {
	if p.TypicalLag > 0 {
		return p.TypicalLag
	}

	return DefaultTypicalLag
}
