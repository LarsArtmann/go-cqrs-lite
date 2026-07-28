package pebbleengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures per-operation costs for the Pebble engine.
// Results feed into PebbleNsPerOp calibration.

func BenchmarkCalibration_PebbleSet(b *testing.B) {
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatalf("NewPebbleEngine: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	defer eng.Close()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}
}

func BenchmarkCalibration_PebbleGet(b *testing.B) {
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatalf("NewPebbleEngine: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	defer eng.Close()

	ctx := context.Background()

	// Pre-populate.
	for i := 0; i < 1000; i++ {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = mb.MapGet(ctx, "bench", i%1000)
	}
}
