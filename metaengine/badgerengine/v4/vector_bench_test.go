package badgerengine_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// LSM brute-force k-NN validation point for the vector-search-at-scale
// spike (docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md) — the
// badger twin of pebbleengine's vector_bench_test.go. VectorInsert returns
// after the write is accepted (wal-synced), so setup at 10K is fast; search
// is the measured O(N·D) prefix scan + distance compute.

const vectorBenchDims = 128

// art-dupl:accept dep-isolated engine module — intentional mirror of the pebbleengine vector bench
func benchBadgerInsertVectors(b *testing.B, vb metaengine.VectorBackend, col string, n int) {
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

// art-dupl:accept dep-isolated engine module — intentional mirror of the pebbleengine vector bench
func benchmarkBadgerVectorSearchAt(b *testing.B, n int) {
	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		b.Skipf("badger not available: %v", err)
	}
	defer func() { _ = eng.Close() }()

	vb := eng.(metaengine.VectorBackend)
	const col = "bench_vec_scale_badger"
	benchBadgerInsertVectors(b, vb, col, n)

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

func BenchmarkBadgerVectorSearch_1K(b *testing.B)  { benchmarkBadgerVectorSearchAt(b, 1_000) }
func BenchmarkBadgerVectorSearch_10K(b *testing.B) { benchmarkBadgerVectorSearchAt(b, 10_000) }
