package dgraphengine_test

import (
	"os"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestRealProfile_ReadCostsPinned pins the SHIPPED per-pattern read costs of
// the live dgraph profile (skip-guarded: needs a reachable Dgraph, e.g. via
// nix run .#ephemeral-dgraph with DGRAPH_ADDR set). A recalibration that
// intentionally moves a number updates this pin in the same commit with fresh
// bench medians (Protocol #4, docs/benchmarks/calibration-2026-08-30.md).
//
// Pinned values (2026-09-06, BenchmarkCalibration_DgraphScaled +
// BenchmarkDgraph_CounterGet, ephemeral Dgraph 25.4.0):
//   - point lookup 350_000 ns  (one MapGet RPC, ~344µs measured)
//   - scan / filtered scan 2_200 ns/row (scaled slopes, 100/1K/10K)
//   - aggregate 2_700 ns/row (CounterGet, ADR-0133)
func TestRealProfile_ReadCostsPinned(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	tests := []struct {
		name    string
		pattern metaengine.ReadPattern
		want    float64
	}{
		{"point lookup", metaengine.ReadPointLookup, 350_000},
		{"filtered scan", metaengine.ReadFilteredScan, 2_200},
		{"aggregate", metaengine.ReadAggregate, 2_700},
		{"scan", metaengine.ReadScan, 2_200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eng.Profile().NsForRead(tt.pattern); got != tt.want {
				t.Errorf(
					"NsForRead(%s) = %.0f, want %.0f — recalibrated constant moved; refresh the pin with fresh medians",
					tt.pattern, got, tt.want,
				)
			}
		})
	}

	// Complexity decision pin (07-43 §g2, 2026-09-07): ADTMap point ops are
	// ONE client-visible RPC — the planner must not multiply the per-RPC
	// constant by log2(volume).
	if c := eng.Profile().Supports[metaengine.ADTMap]; c != metaengine.ComplexityO1 {
		t.Errorf("ADTMap complexity = %v, want O1 (one-RPC point-op decision)", c)
	}
}

// TestCalibrationConstantsDump exports the SHIPPED per-pattern ReadCosts for
// scripts/calibration-drift.sh (enable with CALIB_DUMP=1). DSN-guarded via
// the live-server construction skip so the drift script can cover live
// remote windows (07-43 §f26).
//
// art-dupl:accept per-engine dump of own profile constants (dep-isolated modules)
func TestCalibrationConstantsDump(t *testing.T) {
	if os.Getenv("CALIB_DUMP") != "1" {
		t.Skip("set CALIB_DUMP=1 to dump the shipped calibration constants")
	}

	eng := mustNewDgraphEngine(t)
	rc := eng.Profile().ReadCosts

	t.Logf("CALIB|point_lookup|%.0f|1", rc.NsPerPointLookup)
	t.Logf("CALIB|filtered_scan|%.0f|10000", rc.NsPerFilteredScan)
	t.Logf("CALIB|aggregate|%.0f|1000", rc.NsPerAggregate)
	t.Logf("CALIB|scan|%.0f|10000", rc.NsPerScan)
}
