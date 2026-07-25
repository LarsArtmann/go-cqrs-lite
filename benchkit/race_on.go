//go:build race

package benchkit

// raceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
//
// Local copy kept here (not imported from testutil) to preserve benchkit's
// lean dependency budget. The canonical, repo-wide copy is testutil.RaceEnabled
// (see testutil/race_on.go); modules already depending on testutil should use
// that. See AGENTS.md → "Race-aware test thresholds".
const raceEnabled = true
