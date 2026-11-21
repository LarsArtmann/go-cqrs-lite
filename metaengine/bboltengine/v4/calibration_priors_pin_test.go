package bboltengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if BboltNsPerRowFilteredScan != 620 ||
		BboltNsPerRowFullScan != 660 {
		t.Fatalf("bboltengine calibration priors changed: %d, %d",
			BboltNsPerRowFilteredScan, BboltNsPerRowFullScan)
	}
}
