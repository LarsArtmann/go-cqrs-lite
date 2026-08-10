package pebbleengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures per-operation costs for the Pebble engine.
// Results feed into PebbleNsPerOp calibration.

func BenchmarkCalibration_PebbleSet(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkCalibration_PebbleGet(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()

	// Pre-populate.
	for i := range 1000 {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}

	var i int

	for b.Loop() {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}
		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
		i++
	}
}
