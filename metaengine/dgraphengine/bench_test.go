package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkDgraph_MapSet measures the per-op cost of MapSet on the Dgraph engine.
// Used to validate DG_NsPerOp calibration (gRPC round-trip + RAFT consensus).
func BenchmarkDgraph_MapSet(b *testing.B) {
	addr := dgraphAddr()

	eng, err := dgraphengine.New(addr)
	if err != nil {
		b.Skipf("Dgraph not available: %v", err)
	}
	defer eng.Close()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement MapBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_MapGet measures the per-op cost of MapGet on the Dgraph engine.
// Used to validate DG_NsPerRead calibration (index lookup + gRPC response).
func BenchmarkDgraph_MapGet(b *testing.B) {
	addr := dgraphAddr()

	eng, err := dgraphengine.New(addr)
	if err != nil {
		b.Skipf("Dgraph not available: %v", err)
	}
	defer eng.Close()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement MapBackend")
	}

	ctx := context.Background()

	for i := range 1000 {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("pre-populate MapSet %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}

		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
	}
}

// BenchmarkDgraph_CounterIncrement measures the per-op cost of CounterIncrement.
func BenchmarkDgraph_CounterIncrement(b *testing.B) {
	addr := dgraphAddr()

	eng, err := dgraphengine.New(addr)
	if err != nil {
		b.Skipf("Dgraph not available: %v", err)
	}
	defer eng.Close()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement CounterBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cb.CounterIncrement(
			ctx,
			"bench",
			metaengine.Delta{fmt.Sprintf("c%d", i%10): 1},
		); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_CounterGet measures the per-op cost of CounterGet.
func BenchmarkDgraph_CounterGet(b *testing.B) {
	addr := dgraphAddr()

	eng, err := dgraphengine.New(addr)
	if err != nil {
		b.Skipf("Dgraph not available: %v", err)
	}
	defer eng.Close()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement CounterBackend")
	}

	ctx := context.Background()

	for i := range 1000 {
		if err := cb.CounterIncrement(
			ctx,
			"bench",
			metaengine.Delta{fmt.Sprintf("c%d", i): 1},
		); err != nil {
			b.Fatalf("pre-populate CounterIncrement %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		counts, err := cb.CounterGet(ctx, "bench")
		if err != nil {
			b.Fatalf("CounterGet %d: %v", i, err)
		}

		if len(counts) == 0 {
			b.Fatalf("CounterGet %d: expected non-empty counters", i)
		}
	}
}

// BenchmarkDgraph_SetAdd measures the per-op cost of SetAdd on the Dgraph engine.
// Dgraph uses @index(exact) for O(logN) membership checks.
func BenchmarkDgraph_SetAdd(b *testing.B) {
	addr := dgraphAddr()

	eng, err := dgraphengine.New(addr)
	if err != nil {
		b.Skipf("Dgraph not available: %v", err)
	}
	defer eng.Close()

	sb, ok := eng.(metaengine.SetBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement SetBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := sb.SetAdd(ctx, "bench", fmt.Sprintf("item-%d", i)); err != nil {
			b.Fatalf("SetAdd %d: %v", i, err)
		}
	}
}
