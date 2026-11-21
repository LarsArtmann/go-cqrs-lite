package bigtableengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if BigtableNsPerPointLookup != 2_000_000 ||
		BigtableNsPerAggregate != 2_000_000 ||
		BigtableNsPerRowScan != 900 ||
		BigtableNsPerRowFiltered != 900 {
		t.Fatalf("bigtableengine calibration priors changed: %d, %d, %d, %d",
			BigtableNsPerPointLookup, BigtableNsPerAggregate, BigtableNsPerRowScan, BigtableNsPerRowFiltered)
	}
}
