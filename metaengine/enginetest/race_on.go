//go:build race

package enginetest

// raceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
//
// Local copy kept here (not imported from testutil) because enginetest is a
// non-test package within the metaengine module. See AGENTS.md → "Race-aware
// test thresholds".
const raceEnabled = true
