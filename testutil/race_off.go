//go:build !race

package testutil

// RaceEnabled reports whether the race detector is active (`-race`).
// See race_on.go for the full usage note; this is the non-race build (false).
const RaceEnabled = false
