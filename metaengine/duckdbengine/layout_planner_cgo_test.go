//go:build cgo

package duckdbengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
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

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanner")
	}

	// Apply layout BEFORE seeding so writes go to the planned table.
	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Name"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	// Idempotent: applying again with same fields should not error.
	if err := lp.ApplyLayout("layout", []string{"Category"}, []string{"Name"}); err != nil {
		t.Fatalf("ApplyLayout (idempotent): %v", err)
	}

	ctx := context.Background()
	seedDuckDBProducts(t, eng, "layout")

	// Pushdown should work correctly with the planned table.
	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "layout",
		[]metaengine.FilterSpec{{Column: "Category", Op: metaengine.FilterEq, Value: "veg"}},
		&metaengine.SortSpec{Column: "Name", Desc: true}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("with layout: expected 2 veg items, got %d", len(results.Items))
	}

	// Sort by Name DESC: eggplant > carrot.
	first := results.Items[0].(map[string]any)
	if first["Name"] != "eggplant" {
		t.Errorf("with layout: first = %v, want eggplant", first["Name"])
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

	if err := mb.MapSet(
		ctx,
		"maptest",
		"apple",
		Item{Name: "apple", Category: "fruit", Price: 1.50},
	); err != nil {
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

	// Sort by Name (TEXT) to avoid INTEGER type inference on Price.
	if err := lp.ApplyLayout("cursor", []string{"Category"}, []string{"Name"}); err != nil {
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

	// Sort by Name DESC, cursor = "donut", limit 2.
	// Items with Name < "donut": carrot, banana, apple.
	// Top 2 by DESC: carrot, banana.
	results, err := ps.PushdownMapScan(ctx, "cursor", nil,
		&metaengine.SortSpec{Column: "Name", Desc: true}, "donut", 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("cursor pagination: expected 2, got %d", len(results.Items))
	}

	first := results.Items[0].(map[string]any)
	if first["Name"] != "carrot" {
		t.Errorf("cursor: first = %v, want carrot", first["Name"])
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

	// Verify that Plan auto-calls ApplyLayout when the engine implements
	// LayoutPlanner and the query uses FilterOnField/SortOnField.
	type ItemCreated struct {
		Category string
	}

	type CountInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[CountInput, map[string]int64](
			"category_counts_layout",
			metaengine.On(ItemCreated{}, func(e ItemCreated) metaengine.Delta {
				return metaengine.Delta{e.Category: 1}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Verify the engine still implements LayoutPlanner after Plan.
	if _, ok := eng.(metaengine.LayoutPlanner); !ok {
		t.Fatal("engine should implement LayoutPlanner")
	}

	ctx := context.Background()

	if err := store.Apply(ctx, "ItemCreated", ItemCreated{Category: "books"}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "ItemCreated", ItemCreated{Category: "books"}); err != nil {
		t.Fatal(err)
	}

	result, err := metaengine.ExecuteTyped[CountInput, map[string]int64](ctx, store, CountInput{})
	if err != nil {
		t.Fatal(err)
	}

	if result["books"] != 2 {
		t.Errorf("books count: got %d, want 2", result["books"])
	}
}

func TestDuckDBEngine_ApplyLayoutPlan(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer db.Close()

	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Skipf("DuckDB engine not available: %v", err)
	}
	defer eng.Close()

	lpa, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanApplier")
	}

	type ProductView struct {
		Name     string
		Category string
		Price    float64
		Quantity int
	}

	plan := metaengine.BuildColumnarLayoutPlan("products", reflect.TypeOf(ProductView{}))
	if err := lpa.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	if err := mb.MapSet(ctx, "products", "p1", ProductView{
		Name: "apple", Category: "fruit", Price: 1.50, Quantity: 10,
	}); err != nil {
		t.Fatal(err)
	}

	// Verify all columns were extracted with native types.
	var name, category string
	var price float64
	var quantity int64

	err = db.QueryRowContext(ctx,
		`SELECT Name, Category, Price, Quantity FROM meta_planned_products WHERE key = 'p1'`).Scan(&name, &category, &price, &quantity)
	if err != nil {
		t.Fatalf("query columnar table: %v", err)
	}

	if name != "apple" || category != "fruit" || price != 1.50 || quantity != 10 {
		t.Errorf("columnar row: got %q %q %v %d, want apple fruit 1.50 10", name, category, price, quantity)
	}
}

func TestDuckDBEngine_ColumnarLayoutWithPlan(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer db.Close()

	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Skipf("DuckDB engine not available: %v", err)
	}
	defer eng.Close()

	type ProductCreated struct {
		ID       string
		Name     string
		Category string
		Price    float64
		Quantity int
	}

	type ProductView struct {
		Name     string
		Category string
		Price    float64
		Quantity int
	}

	type ProductInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[ProductInput, ProductView](
			"products",
			metaengine.On(ProductCreated{}, func(e ProductCreated) (string, ProductView) {
				return e.ID, ProductView{
					Name:     e.Name,
					Category: e.Category,
					Price:    e.Price,
					Quantity: e.Quantity,
				}
			}),
			metaengine.WithColumnarLayout[ProductView](),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "ProductCreated", ProductCreated{
		ID: "p1", Name: "apple", Category: "fruit", Price: 1.50, Quantity: 10,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(ctx, "ProductCreated", ProductCreated{
		ID: "p2", Name: "banana", Category: "fruit", Price: 0.75, Quantity: 20,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(ctx, "ProductCreated", ProductCreated{
		ID: "p3", Name: "carrot", Category: "veg", Price: 0.99, Quantity: 15,
	}); err != nil {
		t.Fatal(err)
	}

	// Verify the columnar table has all extracted fields.
	var count int
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM meta_planned_products WHERE key IN ('p1', 'p2', 'p3')`).Scan(&count)
	if err != nil {
		t.Fatalf("count columnar rows: %v", err)
	}
	if count != 3 {
		t.Errorf("columnar row count: got %d, want 3", count)
	}

	// Verify accurate native types: Price is REAL, Quantity is INTEGER.
	var price float64
	var quantity int64
	err = db.QueryRowContext(ctx,
		`SELECT Price, Quantity FROM meta_planned_products WHERE key = 'p1'`).Scan(&price, &quantity)
	if err != nil {
		t.Fatalf("query types: %v", err)
	}
	if price != 1.50 || quantity != 10 {
		t.Errorf("p1 types: got price=%v quantity=%d, want 1.50 10", price, quantity)
	}
}

func TestDuckDBEngine_ColumnarAggregation(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer db.Close()

	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Skipf("DuckDB engine not available: %v", err)
	}
	defer eng.Close()

	type ProductCreated struct {
		ID       string
		Name     string
		Category string
		Price    float64
		Quantity int
	}

	type ProductView struct {
		Name     string
		Category string
		Price    float64
		Quantity int
	}

	type ProductInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[ProductInput, ProductView](
			"products_agg",
			metaengine.On(ProductCreated{}, func(e ProductCreated) (string, ProductView) {
				return e.ID, ProductView{
					Name:     e.Name,
					Category: e.Category,
					Price:    e.Price,
					Quantity: e.Quantity,
				}
			}),
			metaengine.WithColumnarLayout[ProductView](),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	items := []ProductCreated{
		{ID: "p1", Name: "apple", Category: "fruit", Price: 1.50, Quantity: 10},
		{ID: "p2", Name: "banana", Category: "fruit", Price: 0.75, Quantity: 20},
		{ID: "p3", Name: "carrot", Category: "veg", Price: 0.99, Quantity: 15},
		{ID: "p4", Name: "donut", Category: "snack", Price: 2.00, Quantity: 5},
	}
	for _, item := range items {
		if err := store.Apply(ctx, "ProductCreated", item); err != nil {
			t.Fatal(err)
		}
	}

	// Run a vectorized GROUP BY directly on the columnar table. This is the
	// killer feature of columnar-native storage: DuckDB aggregates native
	// columns without decoding JSON blobs.
	rows, err := db.QueryContext(ctx,
		`SELECT Category, COUNT(*), SUM(Price), AVG(Quantity) FROM meta_planned_products_agg GROUP BY Category ORDER BY Category`)
	if err != nil {
		t.Fatalf("aggregation query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	results := make(map[string]struct{ count int; sumPrice float64; avgQuantity float64 })
	for rows.Next() {
		var category string
		var count int
		var sumPrice float64
		var avgQuantity float64
		if err := rows.Scan(&category, &count, &sumPrice, &avgQuantity); err != nil {
			t.Fatal(err)
		}
		results[category] = struct{ count int; sumPrice float64; avgQuantity float64 }{
			count: count, sumPrice: sumPrice, avgQuantity: avgQuantity,
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	fruit, ok := results["fruit"]
	if !ok {
		t.Fatal("missing fruit aggregation")
	}
	if fruit.count != 2 || fruit.sumPrice != 2.25 || fruit.avgQuantity != 15.0 {
		t.Errorf("fruit aggregation: got %+v, want count=2 sumPrice=2.25 avgQuantity=15", fruit)
	}

	veg, ok := results["veg"]
	if !ok {
		t.Fatal("missing veg aggregation")
	}
	if veg.count != 1 || veg.sumPrice != 0.99 || veg.avgQuantity != 15.0 {
		t.Errorf("veg aggregation: got %+v, want count=1 sumPrice=0.99 avgQuantity=15", veg)
	}

	snack, ok := results["snack"]
	if !ok {
		t.Fatal("missing snack aggregation")
	}
	if snack.count != 1 || snack.sumPrice != 2.00 || snack.avgQuantity != 5.0 {
		t.Errorf("snack aggregation: got %+v, want count=1 sumPrice=2.00 avgQuantity=5", snack)
	}
}

