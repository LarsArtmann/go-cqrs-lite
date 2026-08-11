package mysqlengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func BenchmarkCalibration_MySQLSet(b *testing.B) {
	eng := mustNewMySQLEngine(b)
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

func BenchmarkCalibration_MySQLGet(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()

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

func BenchmarkCalibration_MySQLCounterIncrement(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	cb := eng.(metaengine.CounterBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		key := fmt.Sprintf("k%d", i%100)
		if err := cb.CounterIncrement(ctx, "bench", metaengine.Delta{key: 1}); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
		i++
	}
}
