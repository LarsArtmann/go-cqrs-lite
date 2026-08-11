package system_test

import (
	"os"
	"testing"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4" // registers "badger"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4" // registers "pebble"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"     // registers "postgres"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
)

// TestMain ensures all engine drivers are registered exactly once before any
// integration test runs. Individual test files no longer need blank imports.
// CGo-gated drivers (duckdb) are tested in system/integration/, a separate
// Go module that keeps CGo deps out of system/go.mod.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
