//go:build race

package benchkit

// raceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
const raceEnabled = true
