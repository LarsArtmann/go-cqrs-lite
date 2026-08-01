package pgengine_test

import (
	"context"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// seedProducts writes 5 items into the given collection and returns the
// MapBackend for further operations.
func seedProducts(t *testing.T, eng metaengine.Engine, col string) metaengine.MapBackend {
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

	return mb
}

func TestPostgresEngine_PushdownFilter(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "push_filter")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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
}

func TestPostgresEngine_PushdownSort(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "push_sort")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	// Sort by Price descending, limit 3.
	results, err := ps.PushdownMapScan(ctx, "push_sort", nil,
		&metaengine.SortSpec{Column: "Price", Desc: true}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("sort desc limit 3: expected 3, got %d", len(results.Items))
	}

	first := results.Items[0].(map[string]any)
	if first["Name"] != "donut" {
		t.Errorf("desc: first = %v, want donut (price 2.00)", first["Name"])
	}
}

func TestPostgresEngine_PushdownFilterSortLimit(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "push_fsl")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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
}

func TestPostgresEngine_PushdownCursor(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "push_cursor")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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
}

func TestPostgresEngine_PushdownFilterIn(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "push_in")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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

	if len(results.Items) != 2 {
		t.Fatalf("filter IN: expected 3, got %d", len(results.Items))
	}
}

func TestPostgresEngine_ApplyLayout(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedProducts(t, eng, "layout")

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	// Apply layout: expression indexes on Category (filter) and Price (sort).
	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	// Idempotent: applying again should not error.
	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout (idempotent): %v", err)
	}

	// Pushdown should still work correctly with indexes in place.
	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "layout",
		[]metaengine.FilterSpec{{Column: "Category", Op: metaengine.FilterEq, Value: "veg"}},
		&metaengine.SortSpec{Column: "Price", Desc: true}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("with layout: expected 2 veg items, got %d", len(results.Items))
	}

	first := results.Items[0].(map[string]any)
	if first["Name"] != "eggplant" {
		t.Errorf("with layout: first = %v, want eggplant (price 1.25)", first["Name"])
	}
}
