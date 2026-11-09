//go:build !race

package systemscenario

// RaceEnabled reports whether the race detector is active. Exported so a
// consumer writing its own Await budget can relax it the same way.
// See race_on.go.
const RaceEnabled = false
