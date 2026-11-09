package metaengine

import (
	"context"
	"log/slog"
)

// Layout-stamp boot wiring (disk-truth plan A15): layout plans are stamped
// ON by default — every applied plan's fingerprint is compared against the
// one recorded for its collection, drift logs a WARN (warn-first per the
// established G4 pattern; the drift becomes a hard error at v5), and the
// fresh fingerprint is recorded after the successful apply. Operators can
// opt out with WithLayoutStampsDisabled.

// stampLayoutPlan records and drift-checks one applied layout plan against
// the engine's stamp collection (ADR-0143 map persistence — engine resets
// clear stamps with the other materialized state; an absent stamp is
// accepted, so existing collections never fail a boot over missing history).
// Engines without MapBackend are skipped: the stamp rides ordinary map
// persistence, there is nowhere to put it. Failures are advisories (the
// stamp is an observability aid, not a correctness requirement) — they WARN,
// never fail the registration.
func (s *Store) stampLayoutPlan(plan LayoutPlan, engine Engine) {
	if s == nil || s.layoutStampsDisabled {
		return
	}

	mb, ok := engine.(MapBackend)
	if !ok {
		return
	}

	ctx := context.Background()

	diffs, err := LayoutStampDiffs(ctx, mb, []LayoutPlan{plan})
	if err != nil {
		slog.Warn("metaengine: layout stamp comparison failed",
			"collection", plan.Collection, "error", err)
		return
	}

	for _, diff := range diffs {
		slog.Warn("metaengine: layout stamp drift — the declared plan changed since the "+
			"last apply; stored rows predate the current schema (rebuild the collection "+
			"to re-derive them; drift becomes a hard error at v5)",
			"collection", diff.Collection,
			"recorded", diff.Recorded,
			"declared", diff.Declared,
		)
	}

	if err := RecordLayoutStamps(ctx, mb, []LayoutPlan{plan}); err != nil {
		slog.Warn("metaengine: layout stamp record failed",
			"collection", plan.Collection, "error", err)
	}
}

// WithLayoutStampsDisabled turns OFF layout-plan stamping (the A15 default
// is ON, warn-first). With stamping on, every applied layout plan records
// its fingerprint in the engine's LayoutStampCollection collection and a
// changed declaration WARNs on the next apply; engines without map
// persistence skip stamping silently. Disabling removes both the drift
// warning and the stamp writes.
func WithLayoutStampsDisabled() planOption {
	return func(c *planConfig) { c.layoutStampsDisabled = true }
}
