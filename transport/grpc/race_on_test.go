//go:build race

package grpc_test

// raceEnabled reports whether the race detector is active. Timing thresholds
// in pub/sub tests relax under -race because instrumentation inflates goroutine
// scheduling latency 5-10x, causing gRPC stream setup to exceed the normal
// 100ms settle delay.
//
// Local test-only copy (not imported from testutil) to avoid adding a
// test-only dependency to transport/grpc's go.mod. See AGENTS.md →
// "Race-aware test thresholds" and benchkit/race_on.go for the same pattern.
const raceEnabled = true
