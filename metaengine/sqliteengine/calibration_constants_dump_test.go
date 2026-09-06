package sqliteengine_test

import (
	"os"
	"testing"
)

// TestCalibrationConstantsDump exports the SHIPPED per-pattern ReadCosts for
// scripts/calibration-drift.sh (enable with CALIB_DUMP=1). The engine
// profile is the single source of truth for calibrated constants: the drift
// script compares fresh benches against exactly the values the planner
// routes on, so the numbers cannot drift between code and gate.
//
//art-dupl:accept per-engine dump of own profile constants (dep-isolated modules)
func TestCalibrationConstantsDump(t *testing.T) {
	if os.Getenv("CALIB_DUMP") != "1" {
		t.Skip("set CALIB_DUMP=1 to dump the shipped calibration constants")
	}

	eng, db := newSQLiteBenchEngine(t)
	t.Cleanup(func() { _ = db.Close() })
	rc := eng.Profile().ReadCosts

	t.Logf("CALIB|point_lookup|%.0f|1", rc.NsPerPointLookup)
	t.Logf("CALIB|filtered_scan|%.0f|10000", rc.NsPerFilteredScan)
	t.Logf("CALIB|aggregate|%.0f|1000", rc.NsPerAggregate)
	t.Logf("CALIB|scan|%.0f|10000", rc.NsPerScan)
}
