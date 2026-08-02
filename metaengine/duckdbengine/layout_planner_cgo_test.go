//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDuckDBEngine_ApplyLayout(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "layout")

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	// Idempotent: applying again with same fields should not error.
	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout (idempotent): %v", err)
	}

	// Pushdown should still work correctly with the planned table.
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

func TestDuckDBEngine_ApplyLayoutConflict(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("conflict", []string{"Category"}, nil); err != nil {
		t.Fatalf("first ApplyLayout: %v", err)
	}

	// Different columns on same collection → ErrLayoutConflict.
	err = lp.ApplyLayout("conflict", []string{"Name"}, nil)
	if err == nil {
		t.Fatal("expected ErrLayoutConflict, got nil")
	}
}

func TestDuckDBEngine_LayoutMapSetGet(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("maptest", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	type Item struct {
		Name     string
		Category string
		Price    float64
	}

	if err := mb.MapSet(ctx, "maptest", "apple", Item{Name: "apple", Category: "fruit", Price: 1.50}); err != nil {
		t.Fatal(err)
	}

	val, found, err := mb.MapGet(ctx, "maptest", "apple")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected apple to exist")
	}

	m := val.(map[string]any)
	if m["Category"] != "fruit" {
		t.Errorf("Category: got %v, want fruit", m["Category"])
	}
	if m["Price"] != 1.50 {
		t.Errorf("Price: got %v, want 1.50", m["Price"])
	}
}

func TestDuckDBEngine_LayoutMapDelete(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("deltest", []string{"Category"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	type Item struct {
		Name     string
		Category string
	}

	if err := mb.MapSet(ctx, "deltest", "x", Item{Name: "x", Category: "veg"}); err != nil {
		t.Fatal(err)
	}

	if err := mb.MapDelete(ctx, "deltest", "x"); err != nil {
		t.Fatal(err)
	}

	_, found, err := mb.MapGet(ctx, "deltest", "x")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected x to be deleted")
	}
}

func TestDuckDBEngine_LayoutPushdownFilterSortLimit(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("fsl", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

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
		if err := mb.MapSet(ctx, "fsl", item.Name, item); err != nil {
			t.Fatal(err)
		}
	}

	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "fsl",
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

func TestDuckDBEngine_LayoutPushdownCursor(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("cursor", []string{"Category"}, []string{"Price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

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
		if err := mb.MapSet(ctx, "cursor", item.Name, item); err != nil {
			t.Fatal(err)
		}
	}

	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "cursor", nil,
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

func TestDuckDBEngine_LayoutPushdownFilterIn(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	if err := lp.ApplyLayout("filterin", []string{"Category"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	type Item struct {
		Name     string
		Category string
	}

	items := []Item{
		{Name: "apple", Category: "fruit"},
		{Name: "banana", Category: "fruit"},
		{Name: "carrot", Category: "veg"},
		{Name: "donut", Category: "snack"},
		{Name: "eggplant", Category: "veg"},
	}

	for _, item := range items {
		if err := mb.MapSet(ctx, "filterin", item.Name, item); err != nil {
			t.Fatal(err)
		}
	}

	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "filterin",
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
}

func TestDuckDBEngine_LayoutMetaenginePlan(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	type ItemView struct {
		Name     string
		Category string
		Price    float64
	}

	type ListInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[ListInput, []ItemView](
			"products_layout",
			metaengine.On(struct {
				Name     string
				Category string
				Price    float64
			}{}, func(e struct {
				Name     string
				Category string
				Price    float64
			}) (string, ItemView) {
				return e.Name, ItemView{
					Name: e.Name, Category: e.Category, Price: e.Price,
				}
			}),
			metaengine.FilterOnField[ItemView]("Category", metaengine.FilterEq),
			metaengine.SortOnField[ItemView]("Price", true),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "ItemCreated", struct {
		Name     string
		Category string
		Price    float64
	}{Name: "apple", Category: "fruit", Price: 1.50}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "ItemCreated", struct {
		Name     string
		Category string
		Price    float64
	}{Name: "carrot", Category: "veg", Price: 0.99}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "ItemCreated", struct {
		Name     string
		Category string
		Price    float64
	}{Name: "eggplant", Category: "veg", Price: 1.25}); err != nil {
		t.Fatal(err)
	}

	results, err := metaengine.ExecuteTyped[ListInput, []ItemView](ctx, store, ListInput{})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 items, got %d", len(results))
	}
}
