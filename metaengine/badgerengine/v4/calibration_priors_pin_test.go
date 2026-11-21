package badgerengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if BadgerNsPerRowFilteredScan != 650 ||
		BadgerNsPerRowAggregate != 165 ||
		BadgerNsPerRowFullScan != 630 {
		t.Fatalf("badgerengine calibration priors changed: %d, %d, %d",
			BadgerNsPerRowFilteredScan, BadgerNsPerRowAggregate, BadgerNsPerRowFullScan)
	}
}
