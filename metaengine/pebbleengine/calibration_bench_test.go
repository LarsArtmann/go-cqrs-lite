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
	var i int

	for b.Loop() {
		_ = mb.MapSet(ctx, "bench", i, i*2)
		i++
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
	for i := range 1000 {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}

	var i int

	for b.Loop() {
		_, _, _ = mb.MapGet(ctx, "bench", i%1000)
		i++
	}
}
