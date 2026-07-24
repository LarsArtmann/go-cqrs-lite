package metaengine

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// EngineProfile describes what an engine can do and at what cost.
type EngineProfile struct {
	Name     string
	Supports map[ADT]Complexity

	// NsPerOp is the calibrated nanoseconds-per-operation cost for this
	// engine. Used by the cost estimator to compute latency estimates.
	// Zero means use the legacy default (100ns).
	NsPerOp float64
}

func (p EngineProfile) SupportsADT(adt ADT) (Complexity, bool) {
	c, ok := p.Supports[adt]

	return c, ok
}

func (p EngineProfile) String() string {
	var parts []string
	for adt, c := range p.Supports {
		parts = append(parts, fmt.Sprintf("%s@%s", adt, c))
	}

	sort.Strings(parts)

	return fmt.Sprintf("%s: %s", p.Name, strings.Join(parts, " "))
}

// Per-ADT backend interfaces (ISP — engines implement only what they support).

type MapBackend interface {
	MapSet(ctx context.Context, collection string, key any, value any) error
	MapGet(ctx context.Context, collection string, key any) (any, bool, error)
	MapDelete(ctx context.Context, collection string, key any) error
}

// MapUpdater is an optional capability for atomic read-modify-write on map entries.
type MapUpdater interface {
	MapUpdate(ctx context.Context, collection string, key any, update func(prev any) any) error
}

// ScanBackend handles filtered+sorted scans for collection queries.
type ScanBackend interface {
	MapScan(
		ctx context.Context,
		collection string,
		filters []filterPredicate,
		sortFunc func(a, b any) int,
		cursor any,
		limit int,
	) ([]any, error)
}

type SetBackend interface {
	SetAdd(ctx context.Context, collection string, key any) error
	SetContains(ctx context.Context, collection string, key any) (bool, error)
}

type CounterBackend interface {
	CounterIncrement(ctx context.Context, collection string, deltas Delta) error
	CounterGet(ctx context.Context, collection string) (map[string]int64, error)
}

type GraphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// MultimapBackend handles one-to-many key-to-values collections.
type MultimapBackend interface {
	MultiAdd(ctx context.Context, collection string, key any, value any) error
	MultiGet(ctx context.Context, collection string, key any) ([]any, error)
}

// LogBackend handles append-only, ordered log collections.
type LogBackend interface {
	LogAppend(ctx context.Context, collection string, value any) error
	LogTail(ctx context.Context, collection string, limit int) ([]any, error)
}

// Closer is the lifecycle interface.
type Closer interface {
	Close() error
}

// Engine is a storage backend with a cost profile.
type Engine interface {
	Profile() EngineProfile
	Closer
}

// Compile-time assertions that memoryEngine implements all backend interfaces.
var (
	_ Engine          = (*memoryEngine)(nil)
	_ MapBackend      = (*memoryEngine)(nil)
	_ MapUpdater      = (*memoryEngine)(nil)
	_ ScanBackend     = (*memoryEngine)(nil)
	_ SetBackend      = (*memoryEngine)(nil)
	_ CounterBackend  = (*memoryEngine)(nil)
	_ GraphBackend    = (*memoryEngine)(nil)
	_ MultimapBackend = (*memoryEngine)(nil)
	_ LogBackend      = (*memoryEngine)(nil)
)

// SQLiteEngineProfile returns the cost profile for a SQLite engine.
func SQLiteEngineProfile() EngineProfile {
	return EngineProfile{
		Name:   "sqlite",
		NsPerOp: SQLiteNsPerOp,
		Supports: map[ADT]Complexity{
			ADTMap:       ComplexityOLogN,
			ADTSet:       ComplexityOLogN,
			ADTCounter:   ComplexityO1,
			ADTGraph:     ComplexityON,
			ADTSortedMap: ComplexityOLogN,
			ADTLog:       ComplexityOLogN,
			ADTMultimap:  ComplexityOLogN,
		},
	}
}

// SQLiteNsPerOp is the calibrated per-operation cost for the SQLite engine.
// Calibrated via BenchmarkCalibration_SQLiteSet/Get on 2026-07-25 using
// in-memory modernc.org/sqlite (file::memory:):
//   - MapSet (INSERT): ~6,548 ns/op
//   - MapGet (SELECT): ~4,960 ns/op
// The value 7,000 ns is conservative for in-memory planning. Disk-backed
// SQLite adds I/O latency (10-50µs per op), but the planner is designed to
// prefer memory engines when they can serve the query at lower cost.
const SQLiteNsPerOp = 7000.0