package integration_test

import (
	"testing"
	"time"
)

// waitForCondition polls cond every 5ms until it returns true or timeout expires.
// Replaces fixed time.Sleep calls with deterministic eventual-consistency checks.
func waitForCondition(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("condition not met within %v", timeout)
}
