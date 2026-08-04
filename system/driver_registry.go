package system

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite" // SQLite driver registered with database/sql

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DriverFactory creates a metaengine.Engine from an EngineConfig.
// Implementations are registered via [RegisterDriver] at init time.
// The context flows from [New] so engines can respect cancellation/timeouts
// during construction (e.g., SQLite pragma execution).
type DriverFactory func(ctx context.Context, cfg EngineConfig) (metaengine.Engine, error)

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
//	    system.RegisterDriver("sqlite", func(ctx context.Context, cfg system.EngineConfig) (metaengine.Engine, error) {
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
		return nil, fmt.Errorf(
			"%w %q (did you import the driver package?)",
			ErrUnknownDriver, name,
		)
	}

	return factory, nil
}

// lookupBusDriver finds a registered bus driver by name.
func lookupBusDriver(name string) (BusDriverFactory, error) {
	busDriverMu.RLock()
	defer busDriverMu.Unlock()

	factory, ok := busDrivers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBusDriver, name)
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

// init registers built-in drivers.
func init() {
	RegisterDriver("memory", func(_ context.Context, _ EngineConfig) (metaengine.Engine, error) {
		return metaengine.NewMemoryEngine(), nil
	})

	RegisterDriver(
		"sqlite",
		func(ctx context.Context, cfg EngineConfig) (metaengine.Engine, error) {
			dsn := cfg.DSN
			if dsn == "" {
				dsn = ":memory:"
			}

			db, err := sql.Open("sqlite", dsn)
			if err != nil {
				return nil, fmt.Errorf("system: open sqlite %q: %w", dsn, err)
			}

			// Apply pragmas if specified.
			for _, pragma := range cfg.Pragmas {
				if _, err := db.ExecContext(ctx, "PRAGMA "+pragma); err != nil {
					_ = db.Close()

					return nil, fmt.Errorf("system: sqlite pragma %q: %w", pragma, err)
				}
			}

			return metaengine.NewSQLiteEngine(db) //nolint:contextcheck // takes *sql.DB
		},
	)

	// Register the built-in gochannel bus driver (in-process pub/sub).
	RegisterBusDriver("gochannel", func(_ BusConfig) (any, error) {
		return newSimpleBus(), nil
	})
}

// createEngineFromDriver looks up the driver and constructs an engine.
// This replaces the hardcoded switch in createEngine.
func createEngineFromDriver(ctx context.Context, cfg EngineConfig) (metaengine.Engine, error) {
	factory, err := lookupDriver(cfg.Driver)
	if err != nil {
		return nil, err
	}

	eng, err := factory(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("system: driver %q create: %w", cfg.Driver, err)
	}

	return eng, nil
}
