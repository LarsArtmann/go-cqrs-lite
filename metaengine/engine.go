package metaengine

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"strings"
	"time"
)

// EngineProfile describes what an engine can do and at what cost.
type EngineProfile struct {
	Name     string
	Supports map[ADT]Complexity

	// Layouts declares the physical storage layout this engine uses for
	// each ADT. When present, the planner uses the cost matrix
	// (ADT × StorageLayout) → Complexity to reason about WHY one engine
	// beats another. When absent, the planner falls back to the Supports
	// map for complexity (backward compatibility).
	Layouts map[ADT]StorageLayout

	// NsPerOp is the calibrated nanoseconds-per-operation cost for this
	// engine. Used by the cost estimator to compute latency estimates.
	// Zero means use the legacy default (100ns).
	NsPerOp float64

	// NsPerRead is the calibrated nanoseconds-per-READ-operation (point lookups,
	// scans). Engines whose read and write costs differ noticeably (e.g. Pebble:
	// fast LSM point reads, slower writes) should set this. When zero, the
	// planner falls back to NsPerOp, preserving backward compatibility.
	NsPerRead float64

	// NsPerWrite is the calibrated nanoseconds-per-WRITE-operation (inserts,
	// updates, folds). When zero, the planner falls back to NsPerOp.
	NsPerWrite float64

	// Replication declares how this engine's data propagates across process
	// boundaries (DDIA Ch5). ReplicationNone (zero value) means single-node:
	// data stays in this process. All current engines are ReplicationNone.
	// Future distributed engines (e.g. Iroh) will declare their replication mode.
	Replication Replication

	// ReplicationLag is the expected delay between a write on one node and it
	// being visible on another (DDIA Ch5: "replication lag"). Zero for local
	// and primary engines. Used for diagnostics, NOT for latency estimation —
	// staleness is a freshness property, not a performance cost.
	ReplicationLag time.Duration

	// NetworkRTT is the typical round-trip time to reach this engine's data
	// (DDIA Ch1). Zero for in-process engines (Memory, SQLite, Pebble, DuckDB).
	// Non-zero for any engine accessed over a network. Used by the cost
	// estimator as an additive fixed latency component — it does NOT scale
	// with query volume.
	NetworkRTT time.Duration
}

// ReadNsPerOp returns the calibrated per-read-operation cost, falling back to
// NsPerOp when NsPerRead is unset (backward compatibility for older engines).
func (p EngineProfile) ReadNsPerOp() float64 {
	if p.NsPerRead > 0 {
		return p.NsPerRead
	}

	return p.NsPerOp
}

// WriteNsPerOp returns the calibrated per-write-operation cost, falling back to
// NsPerOp when NsPerWrite is unset.
func (p EngineProfile) WriteNsPerOp() float64 {
	if p.NsPerWrite > 0 {
		return p.NsPerWrite
	}

	return p.NsPerOp
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

	extras := []string{}
	if p.IsReplicated() {
		extras = append(extras, fmt.Sprintf("replication=%s", p.Replication))
		if p.ReplicationLag > 0 {
			extras = append(extras, fmt.Sprintf("lag=%s", p.ReplicationLag))
		}
	}
	if p.NetworkRTT > 0 {
		extras = append(extras, fmt.Sprintf("rtt=%s", p.NetworkRTT))
	}

	suffix := ""
	if len(extras) > 0 {
		suffix = " (" + strings.Join(extras, ", ") + ")"
	}

	return fmt.Sprintf("%s: %s%s", p.Name, strings.Join(parts, " "), suffix)
}

// Per-ADT backend interfaces (ISP — engines implement only what they support).

type MapBackend interface {
	MapSet(ctx context.Context, collection string, key any, value any) error
	MapGet(ctx context.Context, collection string, key any) (any, bool, error)
	MapDelete(ctx context.Context, collection string, key any) error
}

// MapUpdater is an optional capability for atomic read-modify-write on map entries.
//
// Type contract: the `prev` parameter in the update callback is engine-dependent:
//   - MemoryEngine: preserves the original Go struct type (e.g. UserView).
//   - SQLite engine: returns map[string]any (decoded from JSON).
//
// The fold-based applyFold path automatically reifies prev to the correct type
// via reifyReflect. For direct MapUpdater usage outside folds, either call
// reify[V](prev) manually or use Store.MapUpdateTyped[V] which handles
// reification automatically.
type MapUpdater interface {
	MapUpdate(ctx context.Context, collection string, key any, update func(prev any) any) error
}

// ScanResult holds the outcome of a collection scan, including pagination
// metadata. Items contains at most limit rows (never limit+1). HasMore is true
// when additional rows exist beyond Items, signalling that the caller should
// offer a cursor for the next page.
type ScanResult struct {
	Items   []any
	HasMore bool
}

// RawScanResult is the raw-bytes variant of ScanResult for ScanRawValues.
type RawScanResult struct {
	Items   [][]byte
	HasMore bool
}

// ScanBackend handles filtered+sorted scans for collection queries.
// Engines that cannot push filtering to the database (e.g. Pebble KV) implement
// this interface to receive a combined filter function and sort comparator
// that are applied in Go. filterFn is nil when no filters are declared.
//
// The returned ScanResult.Items is trimmed to at most limit rows. When more
// rows exist, ScanResult.HasMore is true.
type ScanBackend interface {
	MapScan(
		ctx context.Context,
		collection string,
		filterFn func(item any) bool,
		sortFunc func(a, b any) int,
		cursor any,
		limit int,
	) (ScanResult, error)
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
	// FilterIn is a special operator: Value must be a []any of membership
	// values. SQL builders emit "column IN (?, ?, ...)" instead of a
	// binary operator with a single placeholder.
	FilterIn FilterOp = "IN"
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
	) (ScanResult, error)
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

// LayoutPlanner is an optional capability: engines that can create optimized
// table layouts (extracted columns + indexes) for declared filter/sort fields
// implement this interface. Plan() calls ApplyLayout automatically when a query
// uses FilterOnField/SortOnField and the assigned engine implements this
// interface, eliminating the need for manual NewPlannedSQLiteEngine setup.
type LayoutPlanner interface {
	ApplyLayout(collection string, filterFields, sortFields []string) error
}

// LayoutPlanApplier is an optional extension of LayoutPlanner. Engines that
// implement this interface receive the fully-built LayoutPlan (including
// reflection-derived column types from the result type) instead of rebuilding
// it from field names. This enables accurate native types for columnar layouts
// (e.g. DuckDB/INTEGER vs DOUBLE) and all-fields extraction via WithColumnarLayout.
type LayoutPlanApplier interface {
	LayoutPlanner
	ApplyLayoutPlan(plan LayoutPlan) error
}

// RawValueReader is an optional capability: engines that can read a value's raw
// JSON bytes without decoding to any. ExecuteTyped prefers this path for point
// lookups, avoiding the double-decode tax (any → reify → R becomes raw → R,
// cutting 3 JSON operations to 1).
type RawValueReader interface {
	GetRawValue(ctx context.Context, col string, key any) ([]byte, bool, error)
}

// RawScanReader is an optional capability: engines that can scan collection
// values as raw JSON bytes without decoding each row to any. ExecuteTyped
// prefers this path for filtered scans, avoiding the double-decode tax.
type RawScanReader interface {
	ScanRawValues(
		ctx context.Context,
		col string,
		filters []FilterSpec,
		sort *SortSpec,
		cursor any,
		limit int,
	) (RawScanResult, error)
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
	_ VectorBackend   = (*memoryEngine)(nil)
	_ SearchBackend   = (*memoryEngine)(nil)
	_ SpatialBackend  = (*memoryEngine)(nil)
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
