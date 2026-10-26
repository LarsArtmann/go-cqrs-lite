package turso

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"go.uber.org/goleak"
)

// TestMain owns two teardown contracts at once:
//
//   - snaps.Clean removes unused snapshots for this binary (a stale snapshot
//     file must not be silently carried forward), and
//   - goleak fails the suite when a goroutine outlives it: an engine worker, a
//     catch-up drain loop or a forwarder whose Close was skipped is a product
//     leak, and temporal composability (ADR-0136) makes "the effect stopped"
//     a first-class contract rather than a convention.
//
// goleak runs BEFORE the snapshot cleanup so a leak is reported with its
// creation stack instead of being masked by the exit code.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.Cleanup(func(int) { snaps.Clean(m) }))
}
