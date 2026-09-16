//go:build cgo

package duckdbengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkVectorSearch_SQLPushdown measures the DuckDB engine-scored path:
// distance + ORDER BY + LIMIT all pushed down (O(N), vectorized in C++).
func BenchmarkVectorSearch_SQLPushdown(b *testing.B) {
	eng := mustNewDuckEngine(b)
	b.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	vb := eng.(metaengine.VectorBackend)

	for i := range 1000 {
		values := make([]float32, 64)
		for d := range values {
			values[d] = float32((i*7 + d*13) % 97) / 97
		}

		if err := vb.VectorInsert(ctx, "bench",
			metaengine.Embedding{ID: fmt.Sprintf("v%d", i), Values: values}); err != nil {
			b.Fatalf("VectorInsert: %v", err)
		}
	}

	query := make([]float32, 64)
	for d := range query {
		query[d] = 0.5
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := vb.VectorSearch(ctx, "bench", query, 10, "cosine"); err != nil {
			b.Fatalf("VectorSearch: %v", err)
		}
	}
}
