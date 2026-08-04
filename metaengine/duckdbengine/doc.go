// Package duckdbengine provides a DuckDB-backed metaengine Engine.
//
// DuckDB is an embedded columnar (OLAP) database. This engine excels at
// analytical workloads — CounterBackend uses DuckDB's vectorized GROUP BY
// for O(1) aggregate reads, and MapBackend benefits from columnar scans
// for filtered queries.
//
// The engine implements MapBackend, CounterBackend, ScanBackend, PushdownScan,
// and LayoutPlanner. DuckDB's columnar storage gives Counter reads ComplexityO1
// (vectorized aggregation) vs SQLite's row-oriented O(N) scan.
//
// PushdownScan pushes filter/sort into DuckDB WHERE/ORDER BY using json_extract.
// LayoutPlanner creates dedicated planned tables with extracted columns and ART
// indexes, since DuckDB does not support expression indexes on JSON paths.
// After ApplyLayout, planned-table queries use direct column references instead
// of json_extract, enabling DuckDB's zone maps to prune data blocks.
//
// CGo required: this module statically links the DuckDB C++ engine.
// It is isolated in its own Go module so consumers who don't import it
// never need CGo.
//
// Calibrated cost model (see calibration_bench_test.go for measurements):
//
//	DuckDBNsPerOp   = 15_000  (batch multi-VALUES INSERT, measured ~8,950 ns/row)
//	DuckDBNsPerRead =  1_200  (vectorized scan + aggregation, measured 111-810 ns/row)
package duckdbengine
