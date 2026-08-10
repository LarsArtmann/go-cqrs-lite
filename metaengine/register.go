package metaengine

import "context"

// Self-registration of the built-in memory engine. Like all other engines
// (sqlite, pebble, duckdb, etc.), the memory engine registers itself at init
// time so that operators can select it via config without any code changes.
func init() {
	RegisterDriver("memory", func(_ context.Context, _ DriverConfig) (Engine, error) {
		return NewMemoryEngine(), nil
	})
}
