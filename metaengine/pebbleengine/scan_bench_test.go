package pebbleengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkPebbleScanRawValues_FilterIndex benchmarks the filter index path
// (layout plan with declared filter field) at various collection sizes.
func BenchmarkPebbleScanRawValues_FilterIndex(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ctx := context.Background()
			eng, err := pebbleengine.NewPebbleEngine("")
			if err != nil {
				b.Fatal(err)
			}

			defer eng.Close()

			lp := eng.(metaengine.LayoutPlanner)
			if err := lp.ApplyLayout("items", []string{"score"}, nil); err != nil {
				b.Fatal(err)
			}

			mb := eng.(metaengine.MapBackend)
			for i := range n {
				_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
					"score": float64(i),
				})
			}

			rsr := eng.(metaengine.RawScanReader)
			filter := []metaengine.FilterSpec{
				{Column: "score", Op: metaengine.FilterGe, Value: n / 2},
			}
			sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

			b.ResetTimer()

			for range b.N {
				_, _ = rsr.ScanRawValues(ctx, "items", filter, sortSpec, nil, 10)
			}
		})
	}
}

// BenchmarkPebbleScanRawValues_FullScan benchmarks the full scan path
// (no layout plan) at various collection sizes.
func BenchmarkPebbleScanRawValues_FullScan(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ctx := context.Background()
			eng, err := pebbleengine.NewPebbleEngine("")
			if err != nil {
				b.Fatal(err)
			}

			defer eng.Close()

			mb := eng.(metaengine.MapBackend)
			for i := range n {
				_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
					"score": float64(i),
				})
			}

			rsr := eng.(metaengine.RawScanReader)
			sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

			b.ResetTimer()

			for range b.N {
				_, _ = rsr.ScanRawValues(ctx, "items", nil, sortSpec, nil, 10)
			}
		})
	}
}

// BenchmarkPebbleScanRawValues_SortIndex benchmarks the sort index path
// (layout plan with declared sort field) at various collection sizes.
func BenchmarkPebbleScanRawValues_SortIndex(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ctx := context.Background()
			eng, err := pebbleengine.NewPebbleEngine("")
			if err != nil {
				b.Fatal(err)
			}

			defer eng.Close()

			lp := eng.(metaengine.LayoutPlanner)
			if err := lp.ApplyLayout("items", nil, []string{"score"}); err != nil {
				b.Fatal(err)
			}

			mb := eng.(metaengine.MapBackend)
			for i := range n {
				_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
					"score": float64(i),
				})
			}

			rsr := eng.(metaengine.RawScanReader)
			sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

			b.ResetTimer()

			for range b.N {
				_, _ = rsr.ScanRawValues(ctx, "items", nil, sortSpec, nil, 10)
			}
		})
	}
}

// BenchmarkPebbleScanRawValues_CursorPagination100K benchmarks cursor-based
// pagination at 100K items. Measures the cost of fetching page 2 (items 11-20)
// given a cursor from page 1.
func BenchmarkPebbleScanRawValues_CursorPagination100K(b *testing.B) {
	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatal(err)
	}
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	_ = lp.ApplyLayout("items", []string{"score"}, nil)

	mb := eng.(metaengine.MapBackend)

	const n = 100_000

	for i := range n {
		_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"score": float64(i),
		})
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

	lastScore := float64(9) // score of the 10th item (0-indexed)

	b.ResetTimer()

	for range b.N {
		_, _ = rsr.ScanRawValues(ctx, "items",
			[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGt, Value: lastScore}},
			sortSpec, nil, 10)
	}
}
