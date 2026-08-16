package pebbleengine_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// LSM brute-force k-NN validation point for the vector-search-at-scale
// spike (docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md).
//
// VectorInsert syncs on every write, so setup at 100K vectors is
// fsync-bound; 10K validates that search cost = memory compute + prefix
// scan I/O and extrapolates linearly (search is O(N·D) compute with the
// collection read once per query).

const vectorBenchDims = 128

func benchPebbleInsertVectors(b *testing.B, vb metaengine.VectorBackend, col string, n int) {
	b.Helper()

	rng := rand.New(rand.NewPCG(42, 7))
	batch := make([]float32, vectorBenchDims)

	for i := range n {
		for j := range batch {
			batch[j] = rng.Float32()*2 - 1
		}

		if err := vb.VectorInsert(context.Background(), col,
			metaengine.Embedding{ID: fmt.Sprintf("v%d", i), Values: batch}); err != nil {
			b.Fatalf("VectorInsert %d: %v", i, err)
		}
	}
}

func benchmarkPebbleVectorSearchAt(b *testing.B, n int) {
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Skipf("pebble not available: %v", err)
	}
	defer func() { _ = eng.Close() }()

	vb := eng.(metaengine.VectorBackend)
	const col = "bench_vec_scale_pebble"
	benchPebbleInsertVectors(b, vb, col, n)

	rng := rand.New(rand.NewPCG(99, 1))
	q := make([]float32, vectorBenchDims)
	for j := range q {
		q[j] = rng.Float32()*2 - 1
	}

	ctx := context.Background()
	b.ResetTimer()

	for b.Loop() {
		results, err := vb.VectorSearch(ctx, col, q, 10, "cosine")
		if err != nil {
			b.Fatalf("VectorSearch: %v", err)
		}
		if len(results) != 10 {
			b.Fatalf("k=10 got %d", len(results))
		}
	}
}

func BenchmarkPebbleVectorSearch_1K(b *testing.B)  { benchmarkPebbleVectorSearchAt(b, 1_000) }
func BenchmarkPebbleVectorSearch_10K(b *testing.B) { benchmarkPebbleVectorSearchAt(b, 10_000) }
