//go:build race

package testutil

// RaceEnabled reports whether the race detector is active (`-race`).
// Timing, heap, and latency thresholds in tests relax under -race because
// instrumentation inflates allocations and CPU 5-10x.
//
// Usage (pick the larger bound under -race):
//
//	hang := 5 * time.Second
//	if testutil.RaceEnabled {
//	    hang = 30 * time.Second
//	}
//
// This is the canonical, repo-wide copy. Modules that already depend on
// [testutil] (e.g. integration/) should use this. Modules with a lean
// dependency budget that cannot import testutil may copy the two-file
// build-tag idiom locally — see `benchkit/race_on.go` + `race_off.go` and the
// "Race-aware test thresholds" note in AGENTS.md.
const RaceEnabled = true
