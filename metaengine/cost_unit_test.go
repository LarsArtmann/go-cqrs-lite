package metaengine

import (
	"math"
	"testing"
)

// TestEstimateCost_GraphCostIsBranchingToTheDepth locks the graph traversal
// cost formula: nodes visited = branching^depth, NOT branching*depth.
// With the defaults (branching=10, depth=2) that is 100, not 20.
func TestEstimateCost_GraphCostIsBranchingToTheDepth(t *testing.T) {
	ce := estimateCost(ComplexityODegree, 1000, defaultNsPerOp, 0)

	want := math.Pow(defaultGraphBranchingFactor, defaultGraphTraversalDepth)
	if ce.EstimatedOps != want {
		t.Errorf(
			"graph ops = %v, want branching^depth = %v (%d^%d)",
			ce.EstimatedOps, want, defaultGraphBranchingFactor, defaultGraphTraversalDepth,
		)
	}

	if ce.EstimatedOps != 100 {
		t.Errorf("graph ops with defaults = %v, want 100", ce.EstimatedOps)
	}
}

// TestEstimateCost_VolumeDefaultAppliedAndReported pins the silent-default
// contract: a zero/negative volume falls back to 1000, and the fallback is
// visible in the returned estimate (Volume field) so diagnostics can report it.
func TestEstimateCost_VolumeDefaultAppliedAndReported(t *testing.T) {
	for _, volume := range []int64{0, -5} {
		ce := estimateCost(ComplexityON, volume, defaultNsPerOp, 0)
		if ce.Volume != 1000 {
			t.Errorf("estimateCost(volume=%d).Volume = %d, want default 1000", volume, ce.Volume)
		}

		if ce.EstimatedOps != 1000 {
			t.Errorf(
				"estimateCost(volume=%d).EstimatedOps = %v, want 1000",
				volume,
				ce.EstimatedOps,
			)
		}
	}

	// An explicit volume is used as-is, never overridden.
	ce := estimateCost(ComplexityON, 42, defaultNsPerOp, 0)
	if ce.Volume != 42 || ce.EstimatedOps != 42 {
		t.Errorf("explicit volume 42: got Volume=%d ops=%v, want 42/42", ce.Volume, ce.EstimatedOps)
	}
}

func TestFilterSelectivity(t *testing.T) {
	tests := []struct {
		name    string
		filters int
		want    float64
	}{
		{"no filters → full scan", 0, 1.0},
		{"one filter → 10%", 1, 0.1},
		{"two filters → 1%", 2, 0.01},
		{"three filters → 0.1% (at clamp)", 3, 0.001},
		{"four filters → clamped at 0.001", 4, 0.001},
		{"ten filters → clamped at 0.001", 10, 0.001},
	}

	const epsilon = 1e-12

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterSelectivity(tt.filters)
			if math.Abs(got-tt.want) > epsilon {
				t.Errorf("filterSelectivity(%d) = %v, want %v", tt.filters, got, tt.want)
			}
		})
	}
}
