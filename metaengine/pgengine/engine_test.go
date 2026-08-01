package pgengine_test

import (
	"context"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPostgresEngine_MapBackend(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	type Task struct {
		ID    string
		Title string
	}

	if err := mb.MapSet(ctx, "tasks", "t1", Task{ID: "t1", Title: "Buy milk"}); err != nil {
		t.Fatal(err)
	}

	val, found, err := mb.MapGet(ctx, "tasks", "t1")
	if err != nil {
		t.Fatal(err)
	}

	if !found {
		t.Fatal("expected task t1 to exist")
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}

	if m["Title"] != "Buy milk" {
		t.Errorf("title: got %v, want %q", m["Title"], "Buy milk")
	}
}

func TestPostgresEngine_MapDelete(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	if err := mb.MapSet(ctx, "items", "x", map[string]any{"v": float64(1)}); err != nil {
		t.Fatal(err)
	}

	if _, found, _ := mb.MapGet(ctx, "items", "x"); !found {
		t.Fatal("expected item x to exist before delete")
	}

	if err := mb.MapDelete(ctx, "items", "x"); err != nil {
		t.Fatal(err)
	}

	_, found, _ := mb.MapGet(ctx, "items", "x")
	if found {
		t.Fatal("expected item x to be deleted")
	}
}

func TestPostgresEngine_CounterBackend(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		t.Fatal("engine does not implement CounterBackend")
	}

	if err := cb.CounterIncrement(
		ctx,
		"counts",
		metaengine.Delta{"open": 3, "closed": 1},
	); err != nil {
		t.Fatal(err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": 2}); err != nil {
		t.Fatal(err)
	}

	result, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatal(err)
	}

	if result["open"] != 5 {
		t.Errorf("open: got %d, want 5", result["open"])
	}

	if result["closed"] != 1 {
		t.Errorf("closed: got %d, want 1", result["closed"])
	}
}

func TestPostgresEngine_Profile(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	profile := eng.Profile()

	if profile.Name != "postgres" {
		t.Errorf("name: got %q, want %q", profile.Name, "postgres")
	}

	counterC, ok := profile.Supports[metaengine.ADTCounter]
	if !ok {
		t.Fatal("expected ADTCounter support")
	}

	if counterC != metaengine.ComplexityO1 {
		t.Errorf("counter complexity: got %s, want %s", counterC, metaengine.ComplexityO1)
	}

	layout, ok := profile.Layouts[metaengine.ADTCounter]
	if !ok {
		t.Fatal("expected counter layout declaration")
	}

	if layout != metaengine.LayoutRow {
		t.Errorf("counter layout: got %s, want %s", layout, metaengine.LayoutRow)
	}
}

func TestPostgresEngine_MetaenginePlan(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	type ItemCreated struct {
		Category string
		Count    int64
	}

	type CountInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[CountInput, map[string]int64](
			"category_counts",
			metaengine.On(ItemCreated{}, func(e ItemCreated) metaengine.Delta {
				return metaengine.Delta{e.Category: e.Count}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(
		ctx,
		"ItemCreated",
		ItemCreated{Category: "books", Count: 5},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(
		ctx,
		"ItemCreated",
		ItemCreated{Category: "books", Count: 3},
	); err != nil {
		t.Fatal(err)
	}

	result, err := metaengine.ExecuteTyped[CountInput, map[string]int64](ctx, store, CountInput{})
	if err != nil {
		t.Fatal(err)
	}

	if result["books"] != 8 {
		t.Errorf("books count: got %d, want 8", result["books"])
	}
}

func TestPostgresEngine_ScanBackend(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()

	sb, ok := eng.(metaengine.ScanBackend)
	if !ok {
		t.Fatal("engine does not implement ScanBackend")
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

	mb := eng.(metaengine.MapBackend)

	for _, item := range items {
		if err := mb.MapSet(ctx, "products", item.Name, item); err != nil {
			t.Fatal(err)
		}
	}

	// Test 1: No filter, no sort — should return all 5.
	results, err := sb.MapScan(ctx, "products", nil, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 5 {
		t.Fatalf("no filter: expected 5 results, got %d", len(results))
	}

	// Test 2: Filter by category=fruit — should return 2.
	filterFn := func(item any) bool {
		m, ok := item.(map[string]any)
		if !ok {
			return false
		}

		return m["Category"] == "fruit"
	}

	results, err = sb.MapScan(ctx, "products", filterFn, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("filter fruit: expected 2 results, got %d", len(results))
	}

	// Test 3: Sort by price descending.
	priceSort := func(a, b any) int {
		am, _ := a.(map[string]any)
		bm, _ := b.(map[string]any)
		pa, _ := am["Price"].(float64)
		pb, _ := bm["Price"].(float64)

		switch {
		case pa < pb:
			return 1 // descending
		case pa > pb:
			return -1
		default:
			return 0
		}
	}

	results, err = sb.MapScan(ctx, "products", nil, priceSort, nil, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 4 {
		t.Fatalf("limit 3: expected 4 (limit+1 for has-more), got %d", len(results))
	}

	first, _ := results[0].(map[string]any)
	if first["Name"] != "donut" {
		t.Errorf("sorted desc: first item = %v, want donut", first["Name"])
	}

	// Test 4: Keyset pagination — cursor = donut (price 2.00), limit 2.
	// Should return items with price < 2.00: eggplant(1.25), apple(1.50).
	results, err = sb.MapScan(ctx, "products", nil, priceSort, map[string]any{"Price": 2.0}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf("pagination: expected 3 (limit+1 for has-more), got %d", len(results))
	}
}
