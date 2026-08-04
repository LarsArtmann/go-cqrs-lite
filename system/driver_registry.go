package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DriverFactory creates a metaengine.Engine from an EngineConfig.
// Implementations are registered via [RegisterDriver] at init time.
type DriverFactory func(cfg EngineConfig) (metaengine.Engine, error)

// BusDriverFactory creates an event bus from a BusConfig.
type BusDriverFactory func(cfg BusConfig) (any, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]DriverFactory)

	busDriverMu sync.RWMutex
	busDrivers  = make(map[string]BusDriverFactory)
)

// RegisterDriver registers a storage engine driver factory at init time.
// This is the database/sql model: drivers are compiled in (compile-time safety),
// the operator picks which to activate via config (runtime flexibility).
//
// Typical usage in a driver package's init():
//
//	func init() {
//	    system.RegisterDriver("sqlite", func(cfg system.EngineConfig) (metaengine.Engine, error) {
//	        db, err := sql.Open("sqlite", cfg.DSN)
//	        if err != nil { return nil, err }
//	        return metaengine.NewSQLiteEngine(db)
//	    })
//	}
func RegisterDriver(name string, factory DriverFactory) {
	driverMu.Lock()
	defer driverMu.Unlock()

	drivers[name] = factory
}

// RegisterBusDriver registers a bus driver factory at init time.
func RegisterBusDriver(name string, factory BusDriverFactory) {
	busDriverMu.Lock()
	defer busDriverMu.Unlock()

	busDrivers[name] = factory
}

// lookupDriver finds a registered driver by name.
func lookupDriver(name string) (DriverFactory, error) {
	driverMu.RLock()
	defer driverMu.RUnlock()

	factory, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("system: unknown driver %q (did you import the driver package?)", name)
	}

	return factory, nil
}

// lookupBusDriver finds a registered bus driver by name.
func lookupBusDriver(name string) (BusDriverFactory, error) {
	busDriverMu.RLock()
	defer busDriverMu.Unlock()

	factory, ok := busDrivers[name]
	if !ok {
		return nil, fmt.Errorf("system: unknown bus driver %q", name)
	}

	return factory, nil
}

// RegisteredDrivers returns the names of all registered storage drivers.
func RegisteredDrivers() []string {
	driverMu.RLock()
	defer driverMu.RUnlock()

	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}

	return names
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

// init registers the built-in Memory driver.
func init() {
	RegisterDriver("memory", func(_ EngineConfig) (metaengine.Engine, error) {
		return metaengine.NewMemoryEngine(), nil
	})
}

// createEngineFromDriver looks up the driver and constructs an engine.
// This replaces the hardcoded switch in createEngine.
func createEngineFromDriver(ctx context.Context, cfg EngineConfig) (metaengine.Engine, error) {
	factory, err := lookupDriver(cfg.Driver)
	if err != nil {
		return nil, err
	}

	eng, err := factory(cfg)
	if err != nil {
		return nil, fmt.Errorf("system: driver %q create: %w", cfg.Driver, err)
	}

	return eng, nil
}
