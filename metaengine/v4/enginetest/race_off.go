//go:build !race

package enginetest

// RaceEnabled reports whether the race detector is active. Timing and heap
// thresholds in tests relax under -race because instrumentation inflates
// allocations and CPU 5-10x.
//
// This is the canonical copy for the metaengine module. Engine test packages
// and the metaengine_test package reference this instead of duplicating the
// two-file build-tag idiom. Modules outside metaengine with lean dependency
// budgets (benchkit, transport/grpc, idempotency/kvstore) keep local copies;
// see AGENTS.md → "Race-aware test thresholds".
const RaceEnabled = false
