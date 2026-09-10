package system_test

import (
	"testing"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4" // registers "badger"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4" // registers "pebble"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"     // registers "postgres"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
	"go.uber.org/goleak"
)

// TestMain ensures all engine drivers are registered exactly once before any
// integration test runs. Individual test files no longer need blank imports.
// CGo-gated drivers (duckdb) are tested in system/integration/, a separate
// Go module that keeps CGo deps out of system/go.mod.
//
// goleak.VerifyTestMain turns teardown completeness into a CI failure instead
// of a convention: any goroutine still running after the suite ends (a
// background replan loop that survived Shutdown, an engine closer that missed
// a worker) is reported with its creation stack, so temporal-composability
// regressions (revert leaves something running) surface here first.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
