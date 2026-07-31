//go:build race

package metaengine_test

// raceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
//
// Test-file variant (external test package). See AGENTS.md → "Race-aware
// test thresholds". The canonical repo-wide copy is testutil.RaceEnabled.
const raceEnabled = true
