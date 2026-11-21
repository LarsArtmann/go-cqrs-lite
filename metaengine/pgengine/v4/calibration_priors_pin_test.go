package pgengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if PGNsPerPointLookup != 5_000 ||
		PGNsPerRowFiltered != 400 ||
		PGNsPerRowAggregate != 250 ||
		PGNsPerRowFullScan != 800 {
		t.Fatalf("pgengine calibration priors changed: %d, %d, %d, %d",
			PGNsPerPointLookup, PGNsPerRowFiltered, PGNsPerRowAggregate, PGNsPerRowFullScan)
	}
}
