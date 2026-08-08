package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestTypedReader_AggregateFallback tests the Go-side fallback paths for
// all aggregate methods when using a Memory engine (no SQL pushdown).
// Regression test for:
//   - MIN/MAX initialization bugs (result==0 sentinel fails on 0/negative values)
//   - AVG divisor bug (len(rows) vs nonNullCount)
//   - MultiGroupedAggregate AVG bug (per-spec non-null count vs total group count)
//   - aggregatePushdown missing AggregateCount handling (MultiAggregate fallback)

type aggItem struct {
	ID     string
	Status string
	Price  float64
}

func aggItemQuery() metaengine.QueryDecl[struct{}, aggItem] {
	return metaengine.Query[struct{}, aggItem](
		"agg_items",
		metaengine.On(aggItem{}, func(e aggItem) (string, aggItem) {
			return e.ID, e
		}),
	)
}

func TestTypedReader_AggregateFallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, aggItemQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Negative prices (MIN/MAX init regression), zero price (MIN==0 sentinel regression).
	items := []aggItem{
		{ID: "a", Status: "open", Price: 10},
		{ID: "b", Status: "open", Price: 20},
		{ID: "c", Status: "closed", Price: -5},
		{ID: "d", Status: "closed", Price: 0},
		{ID: "e", Status: "open", Price: 30},
	}

	for _, item := range items {
		if err := store.Apply(ctx, "aggItem", item); err != nil {
			t.Fatalf("Apply %s: %v", item.ID, err)
		}
	}

	reader := metaengine.NewReader[aggItem](store, "agg_items")

	t.Run("Count", func(t *testing.T) {
		n, err := reader.Count(ctx, metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Count = %d, want 5", n)
		}
	})

	t.Run("Sum_price", func(t *testing.T) {
		got, err := reader.Sum(ctx, "Price", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Sum: %v", err)
		}
		assertFloat(t, "Sum(price)", got, 55) // 10+20-5+0+30
	})

	t.Run("Min_price_negative_regression", func(t *testing.T) {
		got, err := reader.Min(ctx, "Price", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Min: %v", err)
		}
		// The result==0 sentinel bug would return 0 instead of -5
		assertFloat(t, "Min(price)", got, -5)
	})

	t.Run("Max_price", func(t *testing.T) {
		got, err := reader.Max(ctx, "Price", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Max: %v", err)
		}
		assertFloat(t, "Max(price)", got, 30)
	})

	t.Run("Avg_price", func(t *testing.T) {
		got, err := reader.Avg(ctx, "Price", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Avg: %v", err)
		}
		assertFloat(t, "Avg(price)", got, 11) // 55/5
	})

	t.Run("GroupedCount", func(t *testing.T) {
		got, err := reader.GroupedCount(ctx, "Status", metaengine.WithLimit(0))
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
		got, err := reader.GroupedSum(ctx, "Price", "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("GroupedSum: %v", err)
		}
		assertFloat(t, "GroupedSum[open]", got["open"], 60)     // 10+20+30
		assertFloat(t, "GroupedSum[closed]", got["closed"], -5) // -5+0
	})

	t.Run("GroupedMin_negative_regression", func(t *testing.T) {
		got, err := reader.GroupedMin(ctx, "Price", "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("GroupedMin: %v", err)
		}
		assertFloat(t, "GroupedMin[closed]", got["closed"], -5)
		assertFloat(t, "GroupedMin[open]", got["open"], 10)
	})

	t.Run("GroupedMax", func(t *testing.T) {
		got, err := reader.GroupedMax(ctx, "Price", "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("GroupedMax: %v", err)
		}
		assertFloat(t, "GroupedMax[closed]", got["closed"], 0)
		assertFloat(t, "GroupedMax[open]", got["open"], 30)
	})

	t.Run("GroupedAvg", func(t *testing.T) {
		got, err := reader.GroupedAvg(ctx, "Price", "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("GroupedAvg: %v", err)
		}
		assertFloat(t, "GroupedAvg[open]", got["open"], 20)       // 60/3
		assertFloat(t, "GroupedAvg[closed]", got["closed"], -2.5) // -5/2
	})

	t.Run("MultiAggregate", func(t *testing.T) {
		got, err := reader.MultiAggregate(ctx, []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "Price", Alias: "total"},
			{Fn: metaengine.AggregateAvg, Column: "Price", Alias: "avg_price"},
			{Fn: metaengine.AggregateMin, Column: "Price", Alias: "min_price"},
			{Fn: metaengine.AggregateMax, Column: "Price", Alias: "max_price"},
		}, metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("MultiAggregate: %v", err)
		}
		if got["cnt"] != 5 {
			t.Errorf("MultiAggregate[cnt] = %v, want 5", got["cnt"])
		}
		assertFloat(t, "MultiAggregate[total]", got["total"], 55)
		assertFloat(t, "MultiAggregate[avg_price]", got["avg_price"], 11)
		assertFloat(t, "MultiAggregate[min_price]", got["min_price"], -5)
		assertFloat(t, "MultiAggregate[max_price]", got["max_price"], 30)
	})

	t.Run("MultiGroupedAggregate", func(t *testing.T) {
		rows, err := reader.MultiGroupedAggregate(ctx, []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "Price", Alias: "total"},
			{Fn: metaengine.AggregateAvg, Column: "Price", Alias: "avg_price"},
			{Fn: metaengine.AggregateMin, Column: "Price", Alias: "min_price"},
		}, "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("MultiGroupedAggregate: %v", err)
		}

		got := make(map[string]map[string]float64, len(rows))
		for _, r := range rows {
			got[r.Group] = r.Values
		}

		if got["open"]["cnt"] != 3 {
			t.Errorf("MultiGroupedAggregate[open][cnt] = %v, want 3", got["open"]["cnt"])
		}
		assertFloat(t, "MultiGroupedAggregate[open][total]", got["open"]["total"], 60)
		assertFloat(t, "MultiGroupedAggregate[open][avg_price]", got["open"]["avg_price"], 20)
		assertFloat(t, "MultiGroupedAggregate[open][min_price]", got["open"]["min_price"], 10)

		if got["closed"]["cnt"] != 2 {
			t.Errorf("MultiGroupedAggregate[closed][cnt] = %v, want 2", got["closed"]["cnt"])
		}
		assertFloat(t, "MultiGroupedAggregate[closed][total]", got["closed"]["total"], -5)
		assertFloat(t, "MultiGroupedAggregate[closed][avg_price]",
			got["closed"]["avg_price"], -2.5)
		assertFloat(t, "MultiGroupedAggregate[closed][min_price]",
			got["closed"]["min_price"], -5)
	})

	t.Run("Distinct_status", func(t *testing.T) {
		got, err := reader.Distinct(ctx, "Status", metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Distinct: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Distinct(status) returned %d values, want 2", len(got))
		}
	})
}

func assertFloat(t *testing.T, label string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}
