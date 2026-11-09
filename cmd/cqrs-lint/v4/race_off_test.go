//go:build !race

package main

// race_enabled reports whether the test binary was built with -race.
const raceEnabled = false
