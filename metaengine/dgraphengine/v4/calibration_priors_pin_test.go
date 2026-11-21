package dgraphengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if DG_NsPerPointLookup != 350_000 ||
		DG_NsPerRowScan != 2_200 ||
		DG_NsPerAggregate != 2_700 {
		t.Fatalf("dgraphengine calibration priors changed: %d, %d, %d",
			DG_NsPerPointLookup, DG_NsPerRowScan, DG_NsPerAggregate)
	}
}
