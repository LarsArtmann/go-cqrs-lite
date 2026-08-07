package pgengine_test

import (
	"context"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// seedProducts writes 5 items into the given collection.
func seedProducts(t *testing.T, eng metaengine.Engine, col string) {
	t.Helper()

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	type Item struct {
		Name     string
		Category string
		Price    float64
	}

	items := []Item{
		{Name: "apple", Category: "fruit", Price: 1.50},
		{Name: "banana", Category: "fruit", Price: 0.75},
		{Name: "carrot", Category: "veg", Price: 0.99},
		{Name: "donut", Category: "snack", Price: 2.00},
		{Name: "eggplant", Category: "veg", Price: 1.25},
	}

	for _, item := range items {
		if err := mb.MapSet(ctx, col, item.Name, item); err != nil {
			t.Fatal(err)
		}
	}
}

// newPostgresPushdown returns the engine + PushdownScan for the calling test.
// The engine is closed automatically via t.Cleanup.
func newPostgresPushdown(t *testing.T) (metaengine.Engine, metaengine.PushdownScan) {
	t.Helper()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	return eng, ps
}

func TestPostgresEngine_PushdownFilter(t *testing.T) {
	t.Parallel()

	eng, ps := newPostgresPushdown(t)

	enginetest.RunPushdownTest(t, eng, "push_filter", seedProducts,
		func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan) {
			// Filter: Category = "fruit" → apple, banana.
			results, err := ps.PushdownMapScan(ctx, "push_filter",
				[]metaengine.FilterSpec{{Column: "Category", Op: metaengine.FilterEq, Value: "fruit"}},
				nil, nil, 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(results.Items) != 2 {
				t.Fatalf("filter fruit: expected 2, got %d", len(results.Items))
			}
		})
}

func TestPostgresEngine_PushdownSort(t *testing.T) {
	t.Parallel()

	eng, ps := newPostgresPushdown(t)

	enginetest.RunPushdownTest(t, eng, "push_sort", seedProducts,
		func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan) {
			// Sort by Price descending, limit 3.
			results, err := ps.PushdownMapScan(ctx, "push_sort", nil,
				&metaengine.SortSpec{Column: "Price", Desc: true}, nil, 3)
			if err != nil {
				t.Fatal(err)
			}

			if len(results.Items) != 3 {
				t.Fatalf("sort desc limit 3: expected 3, got %d", len(results.Items))
			}

			first := results.Items[0].(map[string]any)
			if first["Name"] != "donut" {
				t.Errorf("desc: first = %v, want donut (price 2.00)", first["Name"])
			}
		})
}

func TestPostgresEngine_PushdownFilterSortLimit(t *testing.T) {
	t.Parallel()

	eng, ps := newPostgresPushdown(t)

	enginetest.RunPushdownTest(t, eng, "push_fsl", seedProducts,
		func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan) {
			// Filter: Category = "veg", Sort: Price asc, limit 2.
			// veg items: carrot(0.99), eggplant(1.25). Sorted asc: carrot, eggplant.
			results, err := ps.PushdownMapScan(ctx, "push_fsl",
				[]metaengine.FilterSpec{{Column: "Category", Op: metaengine.FilterEq, Value: "veg"}},
				&metaengine.SortSpec{Column: "Price", Desc: false}, nil, 2)
			if err != nil {
				t.Fatal(err)
			}

			if len(results.Items) != 2 {
				t.Fatalf("filter+sort+limit: expected 2, got %d", len(results.Items))
			}

			first := results.Items[0].(map[string]any)
			if first["Name"] != "carrot" {
				t.Errorf("asc: first = %v, want carrot (price 0.99)", first["Name"])
			}
		})
}

func TestPostgresEngine_PushdownCursor(t *testing.T) {
	t.Parallel()

	eng, ps := newPostgresPushdown(t)

	enginetest.RunPushdownTest(t, eng, "push_cursor", seedProducts,
		func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan) {
			// Sort Price desc, cursor = 2.0 (donut's price), limit 2.
			// Items with price < 2.0: apple(1.50), eggplant(1.25), carrot(0.99), banana(0.75).
			// Top 2: apple, eggplant.
			results, err := ps.PushdownMapScan(ctx, "push_cursor", nil,
				&metaengine.SortSpec{Column: "Price", Desc: true}, 2.0, 2)
			if err != nil {
				t.Fatal(err)
			}

			if len(results.Items) != 2 {
				t.Fatalf("cursor pagination: expected 2, got %d", len(results.Items))
			}

			first := results.Items[0].(map[string]any)
			if first["Name"] != "apple" {
				t.Errorf("cursor: first = %v, want apple (price 1.50)", first["Name"])
			}
		})
}

func TestPostgresEngine_PushdownFilterIn(t *testing.T) {
	t.Parallel()

	eng, ps := newPostgresPushdown(t)

	enginetest.RunPushdownTest(t, eng, "push_in", seedProducts,
		func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan) {
			// FilterIn: Category IN ("fruit", "snack") → apple, banana, donut.
			results, err := ps.PushdownMapScan(ctx, "push_in",
				[]metaengine.FilterSpec{{
					Column: "Category", Op: metaengine.FilterIn,
					Value: []any{"fruit", "snack"},
				}},
				nil, nil, 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(results.Items) != 3 {
				t.Fatalf("filter IN: expected 3, got %d", len(results.Items))
			}
		})
}
