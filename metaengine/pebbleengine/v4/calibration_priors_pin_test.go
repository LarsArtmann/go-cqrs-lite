package pebbleengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if PebbleNsPerRowFilteredScan != 830 ||
		PebbleNsPerRowAggregate != 125 ||
		PebbleNsPerRowFullScan != 700 {
		t.Fatalf("pebbleengine calibration priors changed: %d, %d, %d",
			PebbleNsPerRowFilteredScan, PebbleNsPerRowAggregate, PebbleNsPerRowFullScan)
	}
}
