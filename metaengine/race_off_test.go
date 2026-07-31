//go:build !race

package metaengine_test

// raceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
const raceEnabled = false
