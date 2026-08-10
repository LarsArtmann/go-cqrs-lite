package metaengine

import (
	"context"
	"fmt"
	"sync"
)

// DriverConfig carries storage-engine configuration from the operator to a
// DriverFactory. It is the deployment-time config that the system/ package
// parses from YAML/TOML and passes to the registered driver.
type DriverConfig struct {
	DSN     string
	Pragmas []string
}

// DriverFactory creates an Engine from a DriverConfig. Implementations are
// registered via RegisterDriver at init time — the database/sql model.
type DriverFactory func(ctx context.Context, cfg DriverConfig) (Engine, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]DriverFactory)
)

// RegisterDriver registers a storage engine driver factory at init time.
// This is the database/sql model: drivers are compiled in (compile-time
// safety), the operator picks which to activate via config (runtime
// flexibility).
//
// Typical usage in a driver package's init():
//
//	func init() {
//	    metaengine.RegisterDriver("sqlite", func(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
//	        db, err := sql.Open("sqlite", cfg.DSN)
//	        if err != nil { return nil, err }
//	        return sqliteengine.NewSQLiteEngine(db)
//	    })
//	}
func RegisterDriver(name string, factory DriverFactory) {
	driverMu.Lock()
	defer driverMu.Unlock()

	drivers[name] = factory
}

// LookupDriver finds a registered driver by name.
// Returns ErrUnknownDriver if no driver is registered with that name.
func LookupDriver(name string) (DriverFactory, error) {
	driverMu.RLock()
	defer driverMu.RUnlock()

	factory, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf(
			"%w %q (did you import the driver package?)",
			ErrUnknownDriver, name,
		)
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

// ErrUnknownDriver is returned when LookupDriver cannot find a driver.
var ErrUnknownDriver = fmt.Errorf("unknown driver")

func init() {
	RegisterDriver("memory", func(_ context.Context, _ DriverConfig) (Engine, error) {
		return NewMemoryEngine(), nil
	})
}
