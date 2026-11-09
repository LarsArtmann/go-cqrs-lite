package bench_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_pebble_extensions_test.go — extends the migrated benchmarks with Pebble
// engine comparisons. Pebble is pure Go (LSM point reads), no CGo required.

// TestPromise_ParityWithPebble seeds the same events through Memory and
// Memory+Pebble stores, then verifies all queries return identical results.
// This proves Pebble is a valid routing target for the planner.
func TestPromise_ParityWithPebble(t *testing.T) {
	t.Parallel()

	pebStore := planPromiseStore(t,
		[]metaengine.Engine{metaengine.NewMemoryEngine(), newPebbleEngine(t)})
	defer pebStore.Close()

	runEngineParityTest(t, pebStore, "pebble")
}

// BenchmarkMultiQuery_PebbleApplyThroughput measures event ingestion throughput
// when Pebble is in the engine pool alongside Memory. Pebble excels at point
// reads (LSM), so the planner may route Map lookups to it.
func BenchmarkMultiQuery_PebbleApplyThroughput(b *testing.B) {
	runEngineThroughputBenchmark(b, func(tb testing.TB) metaengine.Engine {
		return newPebbleEngine(tb)
	})
}
