package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestTypedReader_PushdownViaSQLite verifies that TypedReader aggregate
// methods take the SQL pushdown path when backed by a SQLite engine
// (which implements AggregateReader), not the Go-side fallback. This is
// the integration test that bridges the consumer API (TypedReader) with
// the engine capability (AggregateReader) — the existing fallback tests
// only exercise the Memory engine (no pushdown).

type pushItem struct {
	ID     string  `json:"id"`
	Status string  `json:"status"`
	Price  float64 `json:"price"`
}

func pushItemQuery() metaengine.QueryDecl[struct{}, pushItem] {
	return metaengine.Query[struct{}, pushItem](
		"push_items",
		metaengine.OnRecord(pushItem{}, func(_ record.Record, e pushItem) (string, pushItem) {
			return e.ID, e
		}),
	)
}

func TestTypedReader_PushdownViaSQLite(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, pushItemQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	items := []pushItem{
		{ID: "a", Status: "open", Price: 10},
		{ID: "b", Status: "open", Price: 20},
		{ID: "c", Status: "closed", Price: -5},
		{ID: "d", Status: "closed", Price: 0},
		{ID: "e", Status: "open", Price: 30},
	}

	for _, item := range items {
		if err := store.Apply(ctx, "pushItem", item); err != nil {
			t.Fatalf("Apply %s: %v", item.ID, err)
		}
	}

	reader := metaengine.NewReader[pushItem](store, "push_items")

	t.Run("Count", func(t *testing.T) {
		n, err := reader.Count(ctx)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Count = %d, want 5", n)
		}
	})

	t.Run("Sum", func(t *testing.T) {
		got, err := reader.Sum(ctx, "price")
		if err != nil {
			t.Fatalf("Sum: %v", err)
		}
		assertPushFloat(t, "Sum", got, 55)
	})

	t.Run("Min", func(t *testing.T) {
		got, err := reader.Min(ctx, "price")
		if err != nil {
			t.Fatalf("Min: %v", err)
		}
		assertPushFloat(t, "Min", got, -5)
	})

	t.Run("Max", func(t *testing.T) {
		got, err := reader.Max(ctx, "price")
		if err != nil {
			t.Fatalf("Max: %v", err)
		}
		assertPushFloat(t, "Max", got, 30)
	})

	t.Run("Avg", func(t *testing.T) {
		got, err := reader.Avg(ctx, "price")
		if err != nil {
			t.Fatalf("Avg: %v", err)
		}
		assertPushFloat(t, "Avg", got, 11)
	})

	t.Run("Count_with_filter", func(t *testing.T) {
		n, err := reader.Count(ctx,
			metaengine.WithFilter("status", metaengine.FilterEq, "open"))
		if err != nil {
			t.Fatalf("Count filtered: %v", err)
		}
		if n != 3 {
			t.Errorf("Count(open) = %d, want 3", n)
		}
	})

	t.Run("Sum_with_filter", func(t *testing.T) {
		got, err := reader.Sum(ctx, "price",
			metaengine.WithFilter("status", metaengine.FilterEq, "open"))
		if err != nil {
			t.Fatalf("Sum filtered: %v", err)
		}
		assertPushFloat(t, "Sum(open)", got, 60)
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
		assertPushFloat(t, "GroupedSum[open]", got["open"], 60)
		assertPushFloat(t, "GroupedSum[closed]", got["closed"], -5)
	})

	t.Run("MultiAggregate", func(t *testing.T) {
		got, err := reader.MultiAggregate(ctx, []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			{Fn: metaengine.AggregateMin, Column: "price", Alias: "min_price"},
		})
		if err != nil {
			t.Fatalf("MultiAggregate: %v", err)
		}
		if got["cnt"] != 5 {
			t.Errorf("cnt = %v, want 5", got["cnt"])
		}
		assertPushFloat(t, "total", got["total"], 55)
		assertPushFloat(t, "min_price", got["min_price"], -5)
	})

	t.Run("MultiGroupedAggregate", func(t *testing.T) {
		rows, err := reader.MultiGroupedAggregate(ctx, []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			{Fn: metaengine.AggregateAvg, Column: "price", Alias: "avg"},
		}, "status")
		if err != nil {
			t.Fatalf("MultiGroupedAggregate: %v", err)
		}
		got := make(map[string]map[string]float64, len(rows))
		for _, r := range rows {
			got[r.Group] = r.Values
		}
		if got["open"]["cnt"] != 3 {
			t.Errorf("open cnt = %v, want 3", got["open"]["cnt"])
		}
		assertPushFloat(t, "open total", got["open"]["total"], 60)
		assertPushFloat(t, "open avg", got["open"]["avg"], 20)
		assertPushFloat(t, "closed avg", got["closed"]["avg"], -2.5)
	})

	t.Run("Distinct", func(t *testing.T) {
		got, err := reader.Distinct(ctx, "status")
		if err != nil {
			t.Fatalf("DistinctValues: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("DistinctValues = %d, want 2", len(got))
		}
	})

	t.Run("ExplainAggregate", func(t *testing.T) {
		sql, args := reader.ExplainAggregate(ctx,
			metaengine.ExplainAggregateOptions{
				Fn:     metaengine.AggregateSum,
				Column: "price",
			})
		if sql == "" {
			t.Error("ExplainAggregate returned empty SQL")
		}
		if len(args) == 0 {
			t.Error("ExplainAggregate returned no args")
		}
	})
}

func assertPushFloat(t *testing.T, label string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}
