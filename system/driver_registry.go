package system

import (
	"context"
	"fmt"
	"sync"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// BusDriverFactory creates an event bus from a BusConfig.
type BusDriverFactory func(cfg BusConfig) (any, error)

var (
	busDriverMu sync.RWMutex
	busDrivers  = make(map[string]BusDriverFactory)
)

// RegisterBusDriver registers a bus driver factory at init time.
func RegisterBusDriver(name string, factory BusDriverFactory) {
	busDriverMu.Lock()
	defer busDriverMu.Unlock()

	busDrivers[name] = factory
}

// lookupBusDriver finds a registered bus driver by name.
func lookupBusDriver(name string) (BusDriverFactory, error) {
	busDriverMu.RLock()
	defer busDriverMu.RUnlock()

	factory, ok := busDrivers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBusDriver, name)
	}

	return factory, nil
}

// RegisteredBusDrivers returns the names of all registered bus drivers.
func RegisteredBusDrivers() []string {
	busDriverMu.RLock()
	defer busDriverMu.RUnlock()

	names := make([]string, 0, len(busDrivers))
	for name := range busDrivers {
		names = append(names, name)
	}

	return names
}

func init() {
	// Register the built-in gochannel bus driver (Watermill GoChannel).
	RegisterBusDriver("gochannel", func(_ BusConfig) (any, error) {
		return watermill.NewEventBus(), nil
	})

	// Engine drivers self-register via blank imports:
	//   metaengine/      → "memory"
	//   sqliteengine/    → "sqlite"
	//   pebbleengine/    → "pebble" (v5 Phase 4)
	//   badgerengine/    → "badger" (v5 Phase 4)
	// etc. Consumers blank-import the driver packages they need.
}

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
