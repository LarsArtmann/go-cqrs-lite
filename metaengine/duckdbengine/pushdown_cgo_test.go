//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func seedDuckDBProducts(t *testing.T, eng metaengine.Engine, col string) metaengine.MapBackend {
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

func TestDuckDBEngine_PushdownFilter(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "push_filter")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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

func TestDuckDBEngine_PushdownSort(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "push_sort")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	results, err := ps.PushdownMapScan(ctx, "push_sort", nil,
		&metaengine.SortSpec{Column: "Price", Desc: true}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 4 {
		t.Fatalf("sort desc limit 3: expected 4 (limit+1 for has-more), got %d", len(results.Items))
	}

	first := results.Items[0].(map[string]any)
	if first["Name"] != "donut" {
		t.Errorf("desc: first = %v, want donut (price 2.00)", first["Name"])
	}
}

func TestDuckDBEngine_PushdownFilterSortLimit(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "push_fsl")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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

func TestDuckDBEngine_PushdownCursor(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "push_cursor")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	results, err := ps.PushdownMapScan(ctx, "push_cursor", nil,
		&metaengine.SortSpec{Column: "Price", Desc: true}, 2.0, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 3 {
		t.Fatalf("cursor pagination: expected 3 (limit+1 for has-more), got %d", len(results.Items))
	}

	first := results.Items[0].(map[string]any)
	if first["Name"] != "apple" {
		t.Errorf("cursor: first = %v, want apple (price 1.50)", first["Name"])
	}
}

func TestDuckDBEngine_PushdownFilterIn(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "push_in")

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

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
}
