package duckdbengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Profile returns the cost profile for this DuckDB engine.
func (e *duckdbEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "duckdb",
		NsPerOp:     DuckDBNsPerOp,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go).
		// DuckDB's read operations span 4000x: a point lookup (full PK scan +
		// JSON decode via database/sql) is ~500x slower than a vectorized
		// aggregation. Without ReadCosts, the planner uses NsPerRead for all
		// patterns, overestimating scans and underestimating point lookups.
		ReadCosts: metaengine.ReadCosts{
			// Measured ~546K ns (BenchmarkDuckDB_MapGet). Set to a conservative
			// 50K: DuckDB is NOT a point-lookup engine, but the PK index on
			// meta_map avoids a full scan. The 50K still makes the planner
			// prefer Memory (500ns) for point lookups by 100x.
			NsPerPointLookup: 50_000,
			// Measured ~454 ns/row (BenchmarkCalibration_DuckDB_PushdownScan).
			// json_extract WHERE pushdown + vectorized scan.
			NsPerFilteredScan: 450,
			// ~418 ns/row (BenchmarkCalibration_DuckDB_CounterGet, ADR-0133):
			// CounterGet over 1K counters — the actual ReadAggregate
			// execution path (ADTCounter queries). The vectorized-SUM cost
			// (~133 ns/row, BenchmarkCalibration_DuckDB_AggregateSum) prices
			// the typed AggregateReader path, which bypasses the planner.
			NsPerAggregate: 420,
			// Measured ~975 ns/row (BenchmarkCalibration_DuckDB_FullScan).
			// Full scan + Go-side JSON decode of all rows.
			NsPerScan: 1_000,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
			metaengine.ADTSet:       metaengine.ComplexityON,
			metaengine.ADTGraph:     metaengine.ComplexityODegree, // native WITH RECURSIVE on meta_graph_edges
			metaengine.ADTLog:       metaengine.ComplexityON,
			metaengine.ADTMultimap:  metaengine.ComplexityON,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
			// ADR-0142 write-side capabilities (claimkit DuckDB dialect):
			// claims scan the (collection, due_at) index; dedup is a PK upsert.
			metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			metaengine.ADTDedup:    metaengine.ComplexityOLogN,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTSet:      true,
			metaengine.ADTLog:      true,
			metaengine.ADTMultimap: true,
			metaengine.ADTVector:   true,
			metaengine.ADTSearch:   true,
			metaengine.ADTSpatial:  true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutColumnar,
			metaengine.ADTCounter:   metaengine.LayoutColumnar,
			metaengine.ADTSortedMap: metaengine.LayoutColumnar,
		},
	}
	e.ApplyCalibration(&p)

	return p
}
