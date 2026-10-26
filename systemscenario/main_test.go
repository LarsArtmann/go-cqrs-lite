package systemscenario

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain catches an SSE (or subscription) stream that a scenario opened and
// never closed: sse.go runs a read loop per stream, so a forgotten Close leaves
// both the loop and its reader behind. Harness-side leaks are exactly as harmful
// as product ones here — they make the next scenario's assertions flaky.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
