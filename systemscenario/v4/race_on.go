//go:build race

package systemscenario

// RaceEnabled reports whether the race detector is active. Exported so a
// consumer writing its own Await budget can relax it the same way.
//
// This is the repo's canonical two-file build-tag idiom (see
// testutil/race_on.go and the "Race-aware test thresholds" note in AGENTS.md),
// copied locally because systemscenario's dependency budget (system, event,
// command, query, id, errorfamily) forbids importing testutil.
//
// The harness itself does NOT read this value — an await budget is the caller's
// decision — but every in-package await uses it so a 5s wire budget does not
// become a coin flip when the detector inflates CPU 5-10x.
const RaceEnabled = true
