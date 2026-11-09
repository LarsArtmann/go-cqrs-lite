package decider

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain makes writer teardown a gate: the flight recorder (flightrecorder.go)
// spawns a goroutine per repository, and a repo that was never closed leaves it
// running. That was previously invisible — the suite asserted decisions, not
// that the machinery had stopped. Temporal composability (ADR-0136) treats
// "the effect stopped" as a contract, and goleak names the creation stack.
//
// The one ignored goroutine is ginkgo's own interrupt handler (signal relay for
// Ctrl-C during a suite run): it is owned by the test framework and outlives
// the suite by design, not a product leak (same carve-out as metaengine's gate).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction(
		"github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2",
	))
}
