package stack

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain catches a preset whose backend Close was skipped: lifecycle.go hands
// the backend close to a goroutine and waits on it, so an unused or partially
// wired stack leaks one per construction. The suite previously asserted that a
// stack booted, not that it shut down.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
