package tursoengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkVectorSearch_LibSQLPushdown measures the embedded-libSQL path:
// k-NN scoring pushed into SQL via vector_distance_cos (O(N), engine-side
// arithmetic). Compare with sqliteengine's BenchmarkVectorSearch_GoScan —
// same corpus shape, opposite execution path.
func BenchmarkVectorSearch_LibSQLPushdown(b *testing.B) {
	eng, err := tursoengine.New("")
	if err != nil {
		b.Skipf("turso not available: %v", err)
	}
	b.Cleanup(func() { _ = eng.Close() })

	if vp, ok := eng.(metaengine.VectorPathReporter); !ok ||
		vp.VectorSearchPath() != metaengine.VectorPathPushdown {
		b.Fatal("engine is not on the libSQL pushdown path")
	}

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
