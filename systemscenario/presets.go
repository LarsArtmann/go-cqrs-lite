package systemscenario

import (
	"path/filepath"
	"testing"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers the "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// engineNamePrimary / engineNameTimers are the deployment's engine pool
// names; instance roles reference them by these keys.
const (
	engineNamePrimary = "primary"
	engineNameTimers  = "timers"
)

// Memory returns the all-in-memory DeploymentConfig: one memory engine for
// source-of-truth and projections plus a dedicated memory "timers" engine,
// so timer-backed scenarios (TimeAdvances) work out of the box. Tests that
// only need a fast boot use this; nothing survives the scenario.
func Memory() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			engineNamePrimary: {Driver: "memory"},
			engineNameTimers:  {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: engineNamePrimary},
			{Role: system.RoleProjections, Engine: engineNamePrimary},
		},
	}
}

// SQLite returns a file-backed SQLite DeploymentConfig: the primary engine
// (journal + projections) lives in a fresh database under t.TempDir() —
// unique per test, cleaned up by the testing framework — while timers run
// on a dedicated memory engine so TimeAdvances stays fast and deterministic.
//
// The file DSN (not shared-cache in-memory) is deliberate: engines own and
// close their *sql.DB, and shared-cache in-memory databases are destroyed
// when the last connection closes, which makes reopen-style assertions
// flaky. Use SQLite when a scenario should exercise real SQL planning,
// durability pragmas, or file-backed restarts.
func SQLite(tb testing.TB) system.DeploymentConfig {
	dsn := filepath.Join(tb.TempDir(), "scenario.db")

	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			engineNamePrimary: {Driver: "sqlite", DSN: dsn, Pragmas: []string{"journal_mode=wal"}},
			engineNameTimers:  {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: engineNamePrimary},
			{Role: system.RoleProjections, Engine: engineNamePrimary},
		},
	}
}
