package system

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RegisterDriverAlias is a backward-compatible wrapper for external code
// that calls system.RegisterDriver. It delegates to metaengine.RegisterDriver.
//
// Deprecated: Use metaengine.RegisterDriver directly.
func RegisterDriver(name string, factory metaengine.DriverFactory) {
	metaengine.RegisterDriver(name, factory)
}

// RegisteredDrivers returns the names of all registered storage drivers.
// Delegates to metaengine.RegisteredDrivers.
func RegisteredDrivers() []string {
	return metaengine.RegisteredDrivers()
}

// DriverFactory is retained for backward compatibility.
//
// Deprecated: Use metaengine.DriverFactory directly.
type DriverFactory = metaengine.DriverFactory

// createEngineFromDriver looks up the driver and constructs an engine.
func createEngineFromDriver(ctx context.Context, cfg EngineConfig) (metaengine.Engine, error) {
	factory, err := metaengine.LookupDriver(cfg.Driver)
	if err != nil {
		return nil, err
	}

	eng, err := factory(ctx, metaengine.DriverConfig{
		DSN:     cfg.DSN,
		Pragmas: cfg.Pragmas,
	})
	if err != nil {
		return nil, fmt.Errorf("system: driver %q create: %w", cfg.Driver, err)
	}

	return eng, nil
}
