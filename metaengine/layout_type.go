package metaengine

// StorageLayout describes the physical storage structure an engine uses
// for a particular ADT. The planner uses this to reason about WHY one
// engine beats another — e.g., a columnar layout wins for Counter (O(1)
// aggregation) while a row layout wins for point lookups (Map).
type StorageLayout string

const (
	// LayoutRow stores data as rows — optimal for point lookups and
	// single-record updates. Used by SQLite (B-Tree), Memory (hash map).
	LayoutRow StorageLayout = "row"

	// LayoutColumnar stores data as columns — optimal for aggregations
	// and scans over subsets of fields. Used by DuckDB (columnar).
	LayoutColumnar StorageLayout = "columnar"

	// LayoutLSM stores data in a log-structured merge tree — optimal for
	// write-heavy workloads with point reads. Used by Pebble.
	LayoutLSM StorageLayout = "lsm"

	// LayoutKV stores data as key-value pairs — optimal for simple
	// point lookups. Used by Memory (hash map), generic KV stores.
	LayoutKV StorageLayout = "kv"
)
