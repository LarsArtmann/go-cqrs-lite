package middleware

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"go.uber.org/goleak"
)

// TestMain owns two teardown contracts: snaps.Clean prunes unused snapshots,
// and goleak fails the suite when a goroutine outlives it (the flight recorder
// writer in recorder.go is the one this module spawns itself; a circuit-breaker
// or retry loop that survived its chain shows up here too).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.Cleanup(func(int) { snaps.Clean(m) }))
}
