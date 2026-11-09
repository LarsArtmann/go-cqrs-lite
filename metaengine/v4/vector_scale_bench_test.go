package metaengine_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Brute-force k-NN scale baseline for the vector-search-at-scale spike
// (docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md).
//
// MemoryVectorIndex is the in-RAM ceiling: pure Go distance math, no I/O.
// LSM engines (pebble/bbolt/badger) pay the same O(N·D) compute on top of a
// prefix scan, so the numbers here bound every brute-force engine.

const benchDims = 128

func benchInsertVectors(b *testing.B, idx *metaengine.MemoryVectorIndex, col string, n int) {
	b.Helper()

	rng := rand.New(rand.NewPCG(42, 7))
	batch := make([]float32, benchDims)

	for i := range n {
		for j := range batch {
			batch[j] = rng.Float32()*2 - 1
		}

		if err := idx.Insert(
			context.Background(),
			col,
			metaengine.Embedding{ID: fmt.Sprintf("v%d", i), Values: batch},
		); err != nil {
			b.Fatalf("Insert %d: %v", i, err)
		}
	}
}

func benchQueryVector() []float32 {
	rng := rand.New(rand.NewPCG(99, 1))
	q := make([]float32, benchDims)
	for j := range q {
		q[j] = rng.Float32()*2 - 1
	}
	return q
}

func benchmarkMemoryVectorSearchAt(b *testing.B, n int) {
	idx := metaengine.NewMemoryVectorIndex()

	const col = "bench_vec_scale"
	benchInsertVectors(b, idx, col, n)

	q := benchQueryVector()
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		results, err := idx.Search(ctx, col, q, 10, "cosine")
		if err != nil {
			b.Fatalf("Search: %v", err)
		}
		if len(results) != 10 {
			b.Fatalf("k=10 got %d", len(results))
		}
	}
}

func BenchmarkMemoryVectorSearch_1K(b *testing.B)  { benchmarkMemoryVectorSearchAt(b, 1_000) }
func BenchmarkMemoryVectorSearch_10K(b *testing.B) { benchmarkMemoryVectorSearchAt(b, 10_000) }
func BenchmarkMemoryVectorSearch_100K(b *testing.B) {
	benchmarkMemoryVectorSearchAt(b, 100_000)
}
