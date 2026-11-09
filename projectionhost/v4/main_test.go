//go:build !integration

package projectionhost_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain turns teardown completeness into a CI failure instead of a
// convention: any goroutine still running after the suite ends (a worker that
// survived Host.Shutdown, a catch-up drain loop, a dead-letter requeue timer)
// is reported with its creation stack. Temporal composability (ADR-0136)
// makes revert correctness a first-class contract — a Host.Reset/Shutdown that
// leaves something running surfaces here first. This mirrors the system
// module's gate (Cordis M-08) for the projectionhost suite.
//
// The integration build keeps its own TestMain (pg_testcontainer_test.go), so
// this one is compiled out under -tags integration.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
