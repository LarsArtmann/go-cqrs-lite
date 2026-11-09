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
//
// The one ignored goroutine is ginkgo's own interrupt handler (signal relay
// for Ctrl-C during a suite run): it is owned by the test framework and
// outlives the suite by design, not a product leak.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction(
		"github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2",
	))
}
