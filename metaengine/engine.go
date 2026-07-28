package metaengine

import (
	"context"
	"fmt"
	"iter"
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
	parts := make([]string, 0, len(p.Supports))
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
// Engines that cannot push filtering to the database (e.g. Pebble KV) implement
// this interface to receive a combined filter function and sort comparator
// that are applied in Go. filterFn is nil when no filters are declared.
type ScanBackend interface {
	MapScan(
		ctx context.Context,
		collection string,
		filterFn func(item any) bool,
		sortFunc func(a, b any) int,
		cursor any,
		limit int,
	) ([]any, error)
}

// FilterOp is a comparison operator for declarative filter specs.
type FilterOp string

const (
	FilterEq FilterOp = "="
	FilterNe FilterOp = "!="
	FilterLt FilterOp = "<"
	FilterLe FilterOp = "<="
	FilterGt FilterOp = ">"
	FilterGe FilterOp = ">="
)

// FilterSpec is a declarative filter that can be pushed down to the database
// engine. Column is a JSON path within the stored value (e.g. "status"),
// producing json_extract(value, '$.status') on SQLite.
type FilterSpec struct {
	Column string
	Op     FilterOp
	Value  any
}

// SortSpec is a declarative sort directive that can be pushed down to the
// database engine. Column is a JSON path within the stored value.
type SortSpec struct {
	Column string
	Desc   bool
}

// PushdownScan is an optional capability: engines that support SQL-level
// filtering, sorting, and limiting implement this interface to avoid loading
// all rows into Go. The executor checks for this interface at runtime and
// prefers it over ScanBackend.MapScan when declarative FilterSpec/SortSpec
// are available (from FilterOnField/SortOnField).
type PushdownScan interface {
	PushdownMapScan(
		ctx context.Context,
		collection string,
		filters []FilterSpec,
		sort *SortSpec,
		cursor any,
		limit int,
	) ([]any, error)
}

// StreamingScan is an optional capability for engines that support streaming
// iteration over collection data without materializing all rows in memory.
// This is critical for large collections that would cause OOM if loaded all at
// once. The iterator yields values one at a time; the caller processes each
// before the next is fetched.
//
// Engines that implement this interface should also implement ScanBackend or
// PushdownScan. The streaming variant is used when the caller explicitly
// requests streaming (e.g., for batch processing or export operations).
type StreamingScan interface {
	StreamScan(
		ctx context.Context,
		collection string,
		filters []FilterSpec,
		sort *SortSpec,
	) iter.Seq2[any, error]
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
		Name:    "sqlite",
		NsPerOp: SQLiteNsPerOp,
		Supports: map[ADT]Complexity{
			ADTMap:     ComplexityOLogN,
			ADTSet:     ComplexityOLogN,
			ADTCounter: ComplexityO1,
			ADTGraph:   ComplexityON,
			// ADTSortedMap: SQLite now supports PushdownMapScan with json_extract()
			// WHERE/ORDER BY/LIMIT, giving O(logN + k) instead of O(NlogN).
			// Without pushdown (closure-based FilterOn/SortOn), the fallback
			// MapScan path remains O(NlogN). The cost model uses O(logN) as the
			// best-case estimate — queries using FilterOnField/SortOnField achieve it.
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
//
// The value 7,000 ns is conservative for in-memory planning. Disk-backed
// SQLite adds I/O latency (10-50µs per op), but the planner is designed to
// prefer memory engines when they can serve the query at lower cost.
const SQLiteNsPerOp = 7000.0
