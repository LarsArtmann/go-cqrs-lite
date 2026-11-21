package duckdbengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if DuckDBNsPerPointLookup != 50_000 ||
		DuckDBNsPerRowFiltered != 450 ||
		DuckDBNsPerRowAggregate != 420 ||
		DuckDBNsPerRowFullScan != 1_000 {
		t.Fatalf("duckdbengine calibration priors changed: %d, %d, %d, %d",
			DuckDBNsPerPointLookup, DuckDBNsPerRowFiltered, DuckDBNsPerRowAggregate, DuckDBNsPerRowFullScan)
	}
}
