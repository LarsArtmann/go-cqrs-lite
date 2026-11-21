package mysqlengine

import "testing"

// TestCalibrationPriorsPinned freezes the shipped ReadCosts priors: the
// planner routes on these numbers, so any change is a deliberate
// re-calibration (run the calibration benches first), never an accident.
func TestCalibrationPriorsPinned(t *testing.T) {
	if MySQLNsPerPointLookup != 5_000 ||
		MySQLNsPerRowFiltered != 400 ||
		MySQLNsPerRowAggregate != 320 ||
		MySQLNsPerRowFullScan != 800 {
		t.Fatalf("mysqlengine calibration priors changed: %d, %d, %d, %d",
			MySQLNsPerPointLookup, MySQLNsPerRowFiltered, MySQLNsPerRowAggregate, MySQLNsPerRowFullScan)
	}
}
