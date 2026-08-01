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

// layoutComplexity returns the read complexity for a given ADT when stored
// in a particular physical layout. This is the universal cost matrix:
// (ADT × StorageLayout) → Complexity.
//
// The matrix encodes structural reasoning: WHY one engine beats another
// for a given query pattern. For example:
//   - (Counter, Columnar) → O(1) because columnar engines can aggregate
//     a column in O(1) via native SUM/COUNT
//   - (Counter, Row) → O(N) because a row engine must scan all rows
//   - (Map, KV) → O(1) because KV stores are hash-based point lookups
func layoutComplexity(adt ADT, layout StorageLayout) Complexity {
	switch adt {
	case ADTMap:
		switch layout {
		case LayoutKV, LayoutRow, LayoutLSM:
			return ComplexityO1
		case LayoutColumnar:
			return ComplexityO1 // columnar can do point lookup via position
		}

	case ADTSet:
		switch layout {
		case LayoutKV, LayoutRow, LayoutLSM:
			return ComplexityO1
		case LayoutColumnar:
			return ComplexityO1
		}

	case ADTCounter:
		switch layout {
		case LayoutColumnar:
			return ComplexityO1 // native aggregation
		case LayoutKV, LayoutRow, LayoutLSM:
			return ComplexityON // must scan to count
		}

	case ADTSortedMap:
		switch layout {
		case LayoutRow, LayoutLSM:
			return ComplexityOLogN // B-Tree range scan
		case LayoutKV:
			return ComplexityON // no native sort
		case LayoutColumnar:
			return ComplexityONLogN // sort required
		}

	case ADTMultimap:
		switch layout {
		case LayoutRow, LayoutLSM:
			return ComplexityOLogN
		case LayoutKV:
			return ComplexityO1
		case LayoutColumnar:
			return ComplexityON
		}

	case ADTGraph:
		switch layout {
		case LayoutKV:
			return ComplexityODegree // adjacency list
		case LayoutRow, LayoutLSM:
			return ComplexityON // edge scan
		case LayoutColumnar:
			return ComplexityON
		}

	case ADTLog:
		switch layout {
		case LayoutLSM, LayoutRow:
			return ComplexityOLogN // append + sequential read
		case LayoutKV:
			return ComplexityO1
		case LayoutColumnar:
			return ComplexityON
		}
	}

	return ComplexityON // default: assume linear scan
}
