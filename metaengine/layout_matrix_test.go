package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestLayoutMatrix_All16Combinations is the permanent regression guard for the
// operator-lever decision matrix (ADR-0124 Layer 4). It asserts that
// SelectLayout picks the intended LayoutOption for every combination of storage
// layout (KV/LSM/Row/Columnar) and operator priority
// (Balanced/ReadSpeed/WriteSpeed/StorageSpace).
//
// The expected decisions are derived from the calibrated cost constants in
// layout_scoring.go (scoreEmbed / scoreNormalize) and the priority weights in
// priority.go (Priority.Weights). Lower weighted score wins; ties resolve to
// Embed (SelectLayout uses strict <).
//
// Two cells are deliberately fragile and documented:
//
//   - LSM × Balanced: Embed wins by a margin of 0.01 (2.99 vs 3.00).
//   - Columnar × ReadSpeed: exact tie (2.65 vs 2.65); Embed wins on tie-break.
//
// If a future recalibration intentionally flips a cell, update expectedLayout
// here in the same change. An accidental flip will fail this test.
func TestLayoutMatrix_All16Combinations(t *testing.T) {
	t.Parallel()

	priorities := []metaengine.Priority{
		metaengine.PriorityBalanced,
		metaengine.PriorityReadSpeed,
		metaengine.PriorityWriteSpeed,
		metaengine.PriorityStorageSpace,
	}

	layouts := []metaengine.StorageLayout{
		metaengine.LayoutKV,
		metaengine.LayoutLSM,
		metaengine.LayoutRow,
		metaengine.LayoutColumnar,
	}

	for _, layout := range layouts {
		for _, priority := range priorities {
			profile := syntheticProfileFor(layout)

			expected := expectedLayout(layout, priority)
			cell := string(layout) + "/" + string(priority)

			t.Run(cell, func(t *testing.T) {
				t.Parallel()

				got, cost := metaengine.SelectLayout(profile, priority)
				if got != expected {
					t.Fatalf(
						"%s: SelectLayout = %s, want %s (embedScore=%.3f normScore=%.3f)",
						cell, got, expected,
						embedScore(layout, priority), normScore(layout, priority),
					)
				}

				t.Logf(
					"%s: %s (embed %.3f vs norm %.3f, margin %.3f, read=%.2f write=%.2f storage=%.2f)",
					cell,
					got,
					embedScore(layout, priority),
					normScore(layout, priority),
					embedScore(layout, priority)-normScore(layout, priority),
					cost.ReadCost,
					cost.WriteCost,
					cost.StorageCost,
				)
			})
		}
	}
}

// expectedLayout is the authoritative expected decision for each cell. Any
// change to the cost model that flips a decision MUST update this table.
func expectedLayout(
	layout metaengine.StorageLayout,
	priority metaengine.Priority,
) metaengine.LayoutOption {
	switch layout {
	case metaengine.LayoutKV, metaengine.LayoutLSM:
		switch priority {
		case metaengine.PriorityBalanced, metaengine.PriorityReadSpeed:
			return metaengine.LayoutEmbed
		default: // WriteSpeed, StorageSpace
			return metaengine.LayoutNormalize
		}
	case metaengine.LayoutRow:
		return metaengine.LayoutNormalize // JOIN is always cheaper on SQL engines
	case metaengine.LayoutColumnar:
		if priority == metaengine.PriorityReadSpeed {
			return metaengine.LayoutEmbed // exact tie → Embed on tie-break
		}
		return metaengine.LayoutNormalize
	default:
		return metaengine.LayoutEmbed
	}
}

// syntheticProfileFor builds a minimal EngineProfile whose Layouts map maps to
// a single representative layout so defaultStorageLayout returns it
// deterministically.
func syntheticProfileFor(layout metaengine.StorageLayout) metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: "synthetic-" + string(layout),
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: layout,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func embedScore(layout metaengine.StorageLayout, priority metaengine.Priority) float64 {
	costs := metaengine.ScoreLayouts(syntheticProfileFor(layout))
	for _, c := range costs {
		if c.Option == metaengine.LayoutEmbed {
			return c.ScoreWeighted(priority.Weights())
		}
	}
	return 0
}

func normScore(layout metaengine.StorageLayout, priority metaengine.Priority) float64 {
	costs := metaengine.ScoreLayouts(syntheticProfileFor(layout))
	for _, c := range costs {
		if c.Option == metaengine.LayoutNormalize {
			return c.ScoreWeighted(priority.Weights())
		}
	}
	return 0
}
