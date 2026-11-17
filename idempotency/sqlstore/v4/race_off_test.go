//go:build !race

package sqlstore_test

// raceEnabled reports whether the race detector is active. Timing thresholds
// in TTL tests relax under -race because instrumentation inflates goroutine
// scheduling latency 5-10x.
//
// Local test-only copy (not imported from testutil) to avoid adding a
// test-only dependency to sqlstore's go.mod. See AGENTS.md →
// "Race-aware test thresholds" for the pattern and the other lean-budget
// packages that keep local copies.
const raceEnabled = false
