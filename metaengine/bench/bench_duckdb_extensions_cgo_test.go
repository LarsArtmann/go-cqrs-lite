//go:build cgo

package bench_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_duckdb_extensions_cgo_test.go — extends the migrated benchmarks with
// DuckDB engine comparisons. CGo-tagged because DuckDB requires the C++ engine.

// TestPromise_ParityWithDuckDB seeds the same events through Memory and
// Memory+DuckDB stores, then verifies all queries return identical results.
// This proves DuckDB is a valid routing target alongside Memory and SQLite.
func TestPromise_ParityWithDuckDB(t *testing.T) {
	t.Parallel()

	duckStore := planPromiseStore(t,
		[]metaengine.Engine{metaengine.NewMemoryEngine(), newDuckDBEngine(t)})
	defer duckStore.Close()

	runEngineParityTest(t, duckStore, "duckdb")
}

// BenchmarkMultiQuery_DuckDBApplyThroughput measures event ingestion
// throughput when DuckDB is in the engine pool alongside Memory. The planner
// routes filtered queries to DuckDB (pushdown) and point-lookups to Memory.
func BenchmarkMultiQuery_DuckDBApplyThroughput(b *testing.B) {
	runEngineThroughputBenchmark(b, func(tb testing.TB) metaengine.Engine {
		return newDuckDBEngine(tb)
	})
}
