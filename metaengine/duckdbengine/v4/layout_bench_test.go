//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// layoutBenchmarkRows is the dataset size for layout benchmarks. Large enough
// to see zone-map pruning, small enough to run fast.
const layoutBenchmarkRows = 5_000

// BenchmarkDuckDB_LayoutPushdownScan measures a filtered scan on a planned
// table (direct column references + ART index). Compare with
// BenchmarkDuckDB_StandardPushdownScan which uses json_extract on meta_map.
// The speedup ratio demonstrates the value of LayoutPlanner.
func BenchmarkDuckDB_LayoutPushdownScan(b *testing.B) {
	eng, err := duckdbengine.New("")
	if err != nil {
		b.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		b.Fatal("engine does not implement LayoutPlanner")
	}

	// Apply layout BEFORE seeding so writes go to the planned table.
	if err := lp.ApplyLayout("layout_bench", []string{"status"}, []string{"amount"}); err != nil {
		b.Fatalf("ApplyLayout: %v", err)
	}

	populateDuckDBEngine(b, eng, "layout_bench", layoutBenchmarkRows)

	pushdown := eng.(metaengine.PushdownScan)
	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := pushdown.PushdownMapScan(ctx, "layout_bench", filters, nil, nil, 0)
		if err != nil {
			b.Fatalf("PushdownMapScan %d: %v", i, err)
		}

		if len(res.Items) == 0 {
			b.Fatalf("PushdownMapScan %d: expected results, got 0", i)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(layoutBenchmarkRows), "rows-scanned")
}

// BenchmarkDuckDB_StandardPushdownScan measures the same filtered scan without
// a layout — DuckDB uses json_extract on meta_map. The ratio between this and
// the layout variant shows the speedup from planned tables.
func BenchmarkDuckDB_StandardPushdownScan(b *testing.B) {
	eng, err := duckdbengine.New("")
	if err != nil {
		b.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	// No layout — standard meta_map path.
	populateDuckDBEngine(b, eng, "std_bench", layoutBenchmarkRows)

	pushdown := eng.(metaengine.PushdownScan)
	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := pushdown.PushdownMapScan(ctx, "std_bench", filters, nil, nil, 0)
		if err != nil {
			b.Fatalf("PushdownMapScan %d: %v", i, err)
		}

		if len(res.Items) == 0 {
			b.Fatalf("PushdownMapScan %d: expected results, got 0", i)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(layoutBenchmarkRows), "rows-scanned")
}

// BenchmarkDuckDB_LayoutPushdownSort measures a filtered + sorted scan on a
// planned table. The sort column has its own ART index, so DuckDB can use an
// index-ordered scan instead of sorting the json_extract output.
func BenchmarkDuckDB_LayoutPushdownSort(b *testing.B) {
	eng, err := duckdbengine.New("")
	if err != nil {
		b.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		b.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("layout_sort", []string{"status"}, []string{"amount"}); err != nil {
		b.Fatalf("ApplyLayout: %v", err)
	}

	populateDuckDBEngine(b, eng, "layout_sort", layoutBenchmarkRows)

	pushdown := eng.(metaengine.PushdownScan)
	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}
	sort := &metaengine.SortSpec{Column: "amount", Desc: true}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := pushdown.PushdownMapScan(ctx, "layout_sort", filters, sort, nil, 10)
		if err != nil {
			b.Fatalf("PushdownMapScan %d: %v", i, err)
		}

		if len(res.Items) != 10 {
			b.Fatalf("PushdownMapScan %d: expected 10, got %d", i, len(res.Items))
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(layoutBenchmarkRows), "rows-scanned")
}
