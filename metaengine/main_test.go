package metaengine_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain turns teardown completeness into a CI failure instead of a
// convention: any goroutine still running after the suite ends (an auto-replan
// loop that survived its Store, a watcher whose Close was skipped, an engine
// prober left behind) is reported with its creation stack. Temporal
// composability (ADR-0136) makes revert correctness a first-class contract —
// a reset that leaves something running surfaces here first. This mirrors the
// system module's gate (Cordis M-08) for the metaengine suite.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
