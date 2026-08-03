package pgengine_test

import (
	"context"
	"fmt"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkPostgres_MapSet measures the per-op cost of MapSet on the Postgres engine.
// Used to validate PG_NsPerOp calibration.
func BenchmarkPostgres_MapSet(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("postgres engine does not implement MapBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkPostgres_MapGet measures the per-op cost of MapGet on the Postgres engine.
// Used to validate PG_NsPerRead calibration.
func BenchmarkPostgres_MapGet(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("postgres engine does not implement MapBackend")
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

// BenchmarkPostgres_CounterIncrement measures the per-op cost of CounterIncrement.
func BenchmarkPostgres_CounterIncrement(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("postgres engine does not implement CounterBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cb.CounterIncrement(ctx, "bench", metaengine.Delta{fmt.Sprintf("c%d", i%10): 1}); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
	}
}

// BenchmarkPostgres_CounterGet measures the per-op cost of CounterGet.
func BenchmarkPostgres_CounterGet(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("postgres engine does not implement CounterBackend")
	}

	ctx := context.Background()

	for i := range 1000 {
		if err := cb.CounterIncrement(ctx, "bench", metaengine.Delta{fmt.Sprintf("c%d", i): 1}); err != nil {
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
