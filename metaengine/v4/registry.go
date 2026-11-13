package metaengine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// DriverConfig carries storage-engine configuration from the operator to a
// DriverFactory. It is the deployment-time config that the system/ package
// parses from YAML/TOML and passes to the registered driver.
type DriverConfig struct {
	DSN     string
	Pragmas []string

	// Priority is the operator's layout-planning objective for this engine
	// (ADR-0124). An empty value resolves to PriorityBalanced. Drivers may
	// consult it to adapt engine-level defaults (e.g., a SQLite engine could
	// prefer normalized layouts under WriteSpeed).
	Priority Priority

	// Durability is the durability tier requested for this engine: strict,
	// normal, or relaxed (see DurabilityTier). Empty (the default) means
	// unspecified — the engine applies its own defaults. Engines that do not
	// implement per-tier behavior fail construction on any non-empty value
	// (see RejectDurabilityTier) rather than silently ignoring the request.
	Durability DurabilityTier

	// MaterializedViews lists operator-declared aggregate accelerations
	// (see MaterializedViewSpec). Engines that cannot serve them fail
	// construction loudly; engines that can (Turso via incremental view
	// maintenance) derive and create the views at construction. Empty (the
	// default) changes nothing.
	MaterializedViews []MaterializedViewSpec

	// MaxOpenConns is the operator's pool ceiling for engines that own a
	// database/sql handle. SQLite-family engines default to
	// MaxOpenConns(1) — writes serialize into WAL ordering, and one long
	// scan blocks every query behind it. Read-heavy deployments (the Kith
	// read-storm class, issue #112) raise it: WAL allows concurrent
	// readers, and writes then queue behind busy_timeout retries. Zero or
	// negative keeps the engine's default (for SQLite that is the
	// deliberate 1). Engines without a pool ignore the field.
	MaxOpenConns int

	// CellVersioning enables BigTable-style versioned cells (ADR-0141) on
	// this engine, optionally bounding retention: nil (the default) leaves
	// the engine unversioned; a non-nil policy enables versioning, and a
	// zero policy versions with keep-everything retention. Engines that
	// cannot version cells must fail construction on any non-nil value
	// via [RejectCellVersioning] rather than silently ignoring the request.
	// Engines that version natively by construction (BigTable) translate
	// the policy into their native retention knob (the GC policy).
	CellVersioning *RetentionPolicy
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

// RegisteredDrivers returns the names of all registered storage drivers in
// sorted order, so introspection output (System.Explain topology, dashboards)
// stays deterministic across boots regardless of map iteration order.
func RegisteredDrivers() []string {
	driverMu.RLock()
	defer driverMu.RUnlock()

	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// ErrUnknownDriver is returned when LookupDriver cannot find a driver.
var ErrUnknownDriver = errors.New("unknown driver")
