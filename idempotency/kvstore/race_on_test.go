//go:build race

package kvstore_test

// raceEnabled reports whether the race detector is active. Timing thresholds
// in TTL tests relax under -race because instrumentation inflates goroutine
// scheduling latency 5-10x, causing 1ms TTL windows to expire before the test
// goroutine even resumes to observe the expiry check.
//
// Local test-only copy (not imported from testutil) to avoid adding a
// test-only dependency to kvstore's go.mod. See AGENTS.md →
// "Race-aware test thresholds" and benchkit/race_on.go for the same pattern.
const raceEnabled = true
