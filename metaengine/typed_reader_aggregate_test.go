package metaengine

import (
	"context"
	"testing"
)

// TestTypedReader_AggregateFallback tests the Go-side fallback paths for
// all aggregate methods when using a Memory engine (no SQL pushdown).
// This is also the regression test for the MIN/MAX initialization bugs
// (result==0 sentinel) and the AVG divisor bug (len(rows) vs nonNullCount).

func testTypedReaderAggregateFallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := NewMemoryEngine()

	// Insert test data: items with status and price.
	// Notably: some have nil prices (AVG/SUM should skip them),
	// some prices are negative (MIN/MAX init bug regression),
	// and one price is 0 (MIN==0 sentinel regression).
	items := []map[string]any{
		{"id": "a", "status": "open", "price": float64(10)},
		{"id": "b", "status": "open", "price": float64(20)},
		{"id": "c", "status": "closed", "price": float64(-5)},
		{"id": "d", "status": "closed", "price": float64(0)},
		{"id": "e", "status": "open", "price": nil}, // nil: skip in AVG
	}

	store := setupTestStore(t, ctx, eng, "items", items)
	reader := NewReader[map[string]any](store, "items")

	t.Run("Count", func(t *testing.T) {
		n, err := reader.Count(ctx)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Count = %d, want 5", n)
		}
	})

	t.Run("Sum_price", func(t *testing.T) {
		got, err := reader.Sum(ctx, "price")
		if err != nil {
			t.Fatalf("Sum: %v", err)
		}
		// 10 + 20 + (-5) + 0 = 25 (nil skipped)
		if got != 25 {
			t.Errorf("Sum(price) = %v, want 25", got)
		}
	})

	t.Run("Min_price_negative_regression", func(t *testing.T) {
		got, err := reader.Min(ctx, "price")
		if err != nil {
			t.Fatalf("Min: %v", err)
		}
		// Min is -5 (the result==0 sentinel bug would return 0)
		if got != -5 {
			t.Errorf("Min(price) = %v, want -5", got)
		}
	})

	t.Run("Max_price", func(t *testing.T) {
		got, err := reader.Max(ctx, "price")
		if err != nil {
			t.Fatalf("Max: %v", err)
		}
		if got != 20 {
			t.Errorf("Max(price) = %v, want 20", got)
		}
	})

	t.Run("Avg_price_skips_nil", func(t *testing.T) {
		got, err := reader.Avg(ctx, "price")
		if err != nil {
			t.Fatalf("Avg: %v", err)
		}
		// Non-nil prices: 10, 20, -5, 0 → avg = 25/4 = 6.25
		// The bug divided by len(rows)=5 → 5.0
		if got != 6.25 {
			t.Errorf("Avg(price) = %v, want 6.25", got)
		}
	})

	t.Run("GroupedCount", func(t *testing.T) {
		got, err := reader.GroupedCount(ctx, "status")
		if err != nil {
			t.Fatalf("GroupedCount: %v", err)
		}
		if got["open"] != 3 {
			t.Errorf("GroupedCount[open] = %v, want 3", got["open"])
		}
		if got["closed"] != 2 {
			t.Errorf("GroupedCount[closed] = %v, want 2", got["closed"])
		}
	})

	t.Run("GroupedSum", func(t *testing.T) {
		got, err := reader.GroupedSum(ctx, "price", "status")
		if err != nil {
			t.Fatalf("GroupedSum: %v", err)
		}
		if got["open"] != 30 {
			t.Errorf("GroupedSum[open] = %v, want 30", got["open"])
		}
		if got["closed"] != -5 {
			t.Errorf("GroupedSum[closed] = %v, want -5", got["closed"])
		}
	})

	t.Run("GroupedMin_negative_regression", func(t *testing.T) {
		got, err := reader.GroupedMin(ctx, "price", "status")
		if err != nil {
			t.Fatalf("GroupedMin: %v", err)
		}
		if got["closed"] != -5 {
			t.Errorf("GroupedMin[closed] = %v, want -5", got["closed"])
		}
		if got["open"] != 10 {
			t.Errorf("GroupedMin[open] = %v, want 10", got["open"])
		}
	})

	t.Run("GroupedMax_all_negative", func(t *testing.T) {
		// "closed" group has prices -5 and 0 → max is 0
		got, err := reader.GroupedMax(ctx, "price", "status")
		if err != nil {
			t.Fatalf("GroupedMax: %v", err)
		}
		if got["closed"] != 0 {
			t.Errorf("GroupedMax[closed] = %v, want 0", got["closed"])
		}
		if got["open"] != 20 {
			t.Errorf("GroupedMax[open] = %v, want 20", got["open"])
		}
	})

	t.Run("GroupedAvg_skips_nil", func(t *testing.T) {
		got, err := reader.GroupedAvg(ctx, "price", "status")
		if err != nil {
			t.Fatalf("GroupedAvg: %v", err)
		}
		// "open": prices 10, 20, nil → avg = 30/2 = 15
		if got["open"] != 15 {
			t.Errorf("GroupedAvg[open] = %v, want 15", got["open"])
		}
		// "closed": prices -5, 0 → avg = -5/2 = -2.5
		if got["closed"] != -2.5 {
			t.Errorf("GroupedAvg[closed] = %v, want -2.5", got["closed"])
		}
	})

	t.Run("MultiAggregate", func(t *testing.T) {
		got, err := reader.MultiAggregate(ctx, []AggregateSpec{
			{Fn: AggregateCount, Alias: "cnt"},
			{Fn: AggregateSum, Column: "price", Alias: "total"},
			{Fn: AggregateAvg, Column: "price", Alias: "avg_price"},
			{Fn: AggregateMin, Column: "price", Alias: "min_price"},
			{Fn: AggregateMax, Column: "price", Alias: "max_price"},
		})
		if err != nil {
			t.Fatalf("MultiAggregate: %v", err)
		}
		if got["cnt"] != 5 {
			t.Errorf("MultiAggregate[cnt] = %v, want 5", got["cnt"])
		}
		if got["total"] != 25 {
			t.Errorf("MultiAggregate[total] = %v, want 25", got["total"])
		}
		if got["avg_price"] != 6.25 {
			t.Errorf("MultiAggregate[avg_price] = %v, want 6.25", got["avg_price"])
		}
		if got["min_price"] != -5 {
			t.Errorf("MultiAggregate[min_price] = %v, want -5", got["min_price"])
		}
		if got["max_price"] != 20 {
			t.Errorf("MultiAggregate[max_price] = %v, want 20", got["max_price"])
		}
	})

	t.Run("MultiGroupedAggregate_AVG_regression", func(t *testing.T) {
		rows, err := reader.MultiGroupedAggregate(ctx, []AggregateSpec{
			{Fn: AggregateCount, Alias: "cnt"},
			{Fn: AggregateSum, Column: "price", Alias: "total"},
			{Fn: AggregateAvg, Column: "price", Alias: "avg_price"},
		}, "status")
		if err != nil {
			t.Fatalf("MultiGroupedAggregate: %v", err)
		}

		got := make(map[string]map[string]float64, len(rows))
		for _, r := range rows {
			got[r.Group] = r.Values
		}

		// "open": 3 rows, prices 10+20+nil = 30, avg = 30/2 = 15 (not 30/3=10)
		if got["open"]["cnt"] != 3 {
			t.Errorf("MultiGroupedAggregate[open][cnt] = %v, want 3", got["open"]["cnt"])
		}
		if got["open"]["total"] != 30 {
			t.Errorf("MultiGroupedAggregate[open][total] = %v, want 30", got["open"]["total"])
		}
		if got["open"]["avg_price"] != 15 {
			t.Errorf("MultiGroupedAggregate[open][avg_price] = %v, want 15 (not 10)",
				got["open"]["avg_price"])
		}

		// "closed": 2 rows, prices -5+0 = -5, avg = -5/2 = -2.5
		if got["closed"]["cnt"] != 2 {
			t.Errorf("MultiGroupedAggregate[closed][cnt] = %v, want 2", got["closed"]["cnt"])
		}
		if got["closed"]["avg_price"] != -2.5 {
			t.Errorf("MultiGroupedAggregate[closed][avg_price] = %v, want -2.5",
				got["closed"]["avg_price"])
		}
	})

	t.Run("Distinct", func(t *testing.T) {
		got, err := reader.Distinct(ctx, "status")
		if err != nil {
			t.Fatalf("Distinct: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Distinct(status) returned %d values, want 2", len(got))
		}
	})
}

// setupTestStore creates a Store with a Memory engine, inserts the given
// items as map[string]any values, and returns the Store.
func setupTestStore(
	t *testing.T,
	ctx context.Context,
	eng Engine,
	collection string,
	items []map[string]any,
) *Store {
	t.Helper()

	store, err := Plan([]Engine{eng},
		Query[struct{}, map[string]any](collection,
			On(struct{}{}, func(_ struct{}) Delta {
				return Delta{}
			}),
		),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	for _, item := range items {
		if err := store.MapSet(ctx, collection, item["id"], item); err != nil {
			t.Fatalf("MapSet %v: %v", item["id"], err)
		}
	}

	return store
}
