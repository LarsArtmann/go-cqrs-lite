package pebbleengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

// BenchmarkLayoutPlanner_FullScan measures a full-collection scan with a Go-level
// filter on 10K items where only 100 match. Every item is decoded and checked.
func BenchmarkLayoutPlanner_FullScan(b *testing.B) {
	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatal(err)
	}
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for i := 0; i < 10_000; i++ {
		status := "inactive"
		if i%100 == 0 {
			status = "active" // 100 active out of 10K
		}

		if err := mb.MapSet(ctx, "bench", fmt.Sprintf("k%d", i), map[string]any{
			"status": status,
			"idx":    i,
		}); err != nil {
			b.Fatal(err)
		}
	}

	rawReader := eng.(metaengine.RawScanReader)
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := rawReader.ScanRawValues(ctx, "bench", filters, nil, nil, 0)
		if err != nil {
			b.Fatal(err)
		}

		if len(results) != 100 {
			b.Fatalf("expected 100 results, got %d", len(results))
		}
	}
}

// BenchmarkLayoutPlanner_IndexedScan measures the same query using a secondary
// index. Only matching items are visited — O(matches) instead of O(all rows).
func BenchmarkLayoutPlanner_IndexedScan(b *testing.B) {
	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatal(err)
	}
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("bench", []string{"status"}, nil); err != nil {
		b.Fatal(err)
	}

	mb := eng.(metaengine.MapBackend)

	for i := 0; i < 10_000; i++ {
		status := "inactive"
		if i%100 == 0 {
			status = "active" // 100 active out of 10K
		}

		if err := mb.MapSet(ctx, "bench", fmt.Sprintf("k%d", i), map[string]any{
			"status": status,
			"idx":    i,
		}); err != nil {
			b.Fatal(err)
		}
	}

	rawReader := eng.(metaengine.RawScanReader)
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := rawReader.ScanRawValues(ctx, "bench", filters, nil, nil, 0)
		if err != nil {
			b.Fatal(err)
		}

		if len(results) != 100 {
			b.Fatalf("expected 100 results, got %d", len(results))
		}
	}
}
