//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type product struct {
	Category string  `json:"category"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

func seedProducts(t *testing.T, eng metaengine.Engine, col string) {
	t.Helper()

	ctx := context.Background()
	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	items := []product{
		{"electronics", "laptop", 1200.0, 5},
		{"electronics", "mouse", 45.0, 20},
		{"electronics", "keyboard", 80.0, 15},
		{"books", "go-book", 35.0, 30},
		{"books", "novel", 12.0, 50},
		{"food", "apple", 1.5, 100},
		{"food", "bread", 3.0, 40},
	}

	for i, p := range items {
		if err := mb.MapSet(ctx, col, p.Name, p); err != nil {
			t.Fatalf("MapSet[%d]: %v", i, err)
		}
	}
}

// --- AggregateReader (scalar aggregates) ---

func TestDuckDB_Aggregate_Count(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_count"
	seedProducts(t, eng, col)

	ctx := context.Background()
	ar, ok := eng.(metaengine.AggregateReader)
	if !ok {
		t.Fatal("engine does not implement AggregateReader")
	}

	n, err := ar.Aggregate(ctx, col, metaengine.AggregateCount, "", nil)
	if err != nil {
		t.Fatalf("Aggregate COUNT: %v", err)
	}

	if n != 7 {
		t.Errorf("Count: got %v, want 7", n)
	}
}

func TestDuckDB_Aggregate_Sum(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_sum"
	seedProducts(t, eng, col)

	ctx := context.Background()
	ar, ok := eng.(metaengine.AggregateReader)
	if !ok {
		t.Fatal("engine does not implement AggregateReader")
	}

	// Sum of prices: 1200+45+80+35+12+1.5+3 = 1376.5
	sum, err := ar.Aggregate(ctx, col, metaengine.AggregateSum, "price", nil)
	if err != nil {
		t.Fatalf("Aggregate SUM: %v", err)
	}

	if sum != 1376.5 {
		t.Errorf("Sum price: got %v, want 1376.5", sum)
	}

	// Sum of quantities: 5+20+15+30+50+100+40 = 260
	sumQty, err := ar.Aggregate(ctx, col, metaengine.AggregateSum, "quantity", nil)
	if err != nil {
		t.Fatalf("Aggregate SUM quantity: %v", err)
	}

	if sumQty != 260 {
		t.Errorf("Sum quantity: got %v, want 260", sumQty)
	}
}

func TestDuckDB_Aggregate_WithFilters(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_filt"
	seedProducts(t, eng, col)

	ctx := context.Background()
	ar, ok := eng.(metaengine.AggregateReader)
	if !ok {
		t.Fatal("engine does not implement AggregateReader")
	}

	// Count electronics: 3
	n, err := ar.Aggregate(
		ctx,
		col,
		metaengine.AggregateCount,
		"",
		[]metaengine.FilterSpec{
			{Column: "category", Op: metaengine.FilterEq, Value: "electronics"},
		},
	)
	if err != nil {
		t.Fatalf("Aggregate COUNT with filter: %v", err)
	}

	if n != 3 {
		t.Errorf("Count electronics: got %v, want 3", n)
	}

	// Sum electronics prices: 1200+45+80 = 1325
	sum, err := ar.Aggregate(
		ctx,
		col,
		metaengine.AggregateSum,
		"price",
		[]metaengine.FilterSpec{
			{Column: "category", Op: metaengine.FilterEq, Value: "electronics"},
		},
	)
	if err != nil {
		t.Fatalf("Aggregate SUM with filter: %v", err)
	}

	if sum != 1325.0 {
		t.Errorf("Sum electronics price: got %v, want 1325", sum)
	}
}

func TestDuckDB_Aggregate_MinMaxAvg(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_mma"
	seedProducts(t, eng, col)

	ctx := context.Background()
	ar := eng.(metaengine.AggregateReader)

	minP, err := ar.Aggregate(ctx, col, metaengine.AggregateMin, "price", nil)
	if err != nil {
		t.Fatalf("MIN: %v", err)
	}

	if minP != 1.5 {
		t.Errorf("Min price: got %v, want 1.5", minP)
	}

	maxP, err := ar.Aggregate(ctx, col, metaengine.AggregateMax, "price", nil)
	if err != nil {
		t.Fatalf("MAX: %v", err)
	}

	if maxP != 1200.0 {
		t.Errorf("Max price: got %v, want 1200", maxP)
	}

	avgP, err := ar.Aggregate(ctx, col, metaengine.AggregateAvg, "price", nil)
	if err != nil {
		t.Fatalf("AVG: %v", err)
	}

	// 1376.5 / 7 = 196.642857...
	expected := 1376.5 / 7
	if avgP < expected-0.01 || avgP > expected+0.01 {
		t.Errorf("Avg price: got %v, want ~%v", avgP, expected)
	}
}

// --- GroupedAggregateReader (GROUP BY + single aggregate) ---

func TestDuckDB_GroupedAggregate_Count(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_gcount"
	seedProducts(t, eng, col)

	ctx := context.Background()
	gr, ok := eng.(metaengine.GroupedAggregateReader)
	if !ok {
		t.Fatal("engine does not implement GroupedAggregateReader")
	}

	result, err := gr.GroupedAggregate(ctx, col, metaengine.AggregateCount, "", "category", nil)
	if err != nil {
		t.Fatalf("GroupedAggregate COUNT: %v", err)
	}

	if result["electronics"] != 3 {
		t.Errorf("electronics count: got %v, want 3", result["electronics"])
	}

	if result["books"] != 2 {
		t.Errorf("books count: got %v, want 2", result["books"])
	}

	if result["food"] != 2 {
		t.Errorf("food count: got %v, want 2", result["food"])
	}
}

func TestDuckDB_GroupedAggregate_Sum(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_gsum"
	seedProducts(t, eng, col)

	ctx := context.Background()
	gr, ok := eng.(metaengine.GroupedAggregateReader)
	if !ok {
		t.Fatal("engine does not implement GroupedAggregateReader")
	}

	result, err := gr.GroupedAggregate(ctx, col, metaengine.AggregateSum, "price", "category", nil)
	if err != nil {
		t.Fatalf("GroupedAggregate SUM: %v", err)
	}

	// electronics: 1200+45+80 = 1325
	if result["electronics"] != 1325.0 {
		t.Errorf("electronics sum: got %v, want 1325", result["electronics"])
	}

	// books: 35+12 = 47
	if result["books"] != 47.0 {
		t.Errorf("books sum: got %v, want 47", result["books"])
	}

	// food: 1.5+3 = 4.5
	if result["food"] != 4.5 {
		t.Errorf("food sum: got %v, want 4.5", result["food"])
	}
}

func TestDuckDB_GroupedAggregate_WithFilters(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_gsum_filt"
	seedProducts(t, eng, col)

	ctx := context.Background()
	gr := eng.(metaengine.GroupedAggregateReader)

	// Only items with price > 10
	result, err := gr.GroupedAggregate(ctx, col, metaengine.AggregateCount, "", "category",
		[]metaengine.FilterSpec{{Column: "price", Op: metaengine.FilterGt, Value: float64(10)}})
	if err != nil {
		t.Fatalf("GroupedAggregate COUNT filtered: %v", err)
	}

	// electronics: all 3 (1200, 45, 80 all > 10)
	if result["electronics"] != 3 {
		t.Errorf("electronics count (price>10): got %v, want 3", result["electronics"])
	}

	// books: both (35, 12 > 10)
	if result["books"] != 2 {
		t.Errorf("books count (price>10): got %v, want 2", result["books"])
	}

	// food: none (1.5 and 3.0 are NOT > 10)
	if _, exists := result["food"]; exists {
		t.Errorf("food should not appear (no items > 10), got %v", result["food"])
	}
}

// --- MultiAggregateReader (multiple scalar aggregates in one pass) ---

func TestDuckDB_MultiAggregate(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_multi"
	seedProducts(t, eng, col)

	ctx := context.Background()
	mr, ok := eng.(metaengine.MultiAggregateReader)
	if !ok {
		t.Fatal("engine does not implement MultiAggregateReader")
	}

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "total"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "revenue"},
		{Fn: metaengine.AggregateMin, Column: "price", Alias: "cheapest"},
		{Fn: metaengine.AggregateMax, Column: "price", Alias: "priciest"},
	}

	result, err := mr.MultiAggregate(ctx, col, specs, nil)
	if err != nil {
		t.Fatalf("MultiAggregate: %v", err)
	}

	if result["total"] != 7 {
		t.Errorf("total: got %v, want 7", result["total"])
	}

	if result["revenue"] != 1376.5 {
		t.Errorf("revenue: got %v, want 1376.5", result["revenue"])
	}

	if result["cheapest"] != 1.5 {
		t.Errorf("cheapest: got %v, want 1.5", result["cheapest"])
	}

	if result["priciest"] != 1200.0 {
		t.Errorf("priciest: got %v, want 1200", result["priciest"])
	}
}

// --- MultiGroupedAggregateReader (GROUP BY + multiple aggregates) ---

func TestDuckDB_MultiGroupedAggregate(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_mg"
	seedProducts(t, eng, col)

	ctx := context.Background()
	mgr, ok := eng.(metaengine.MultiGroupedAggregateReader)
	if !ok {
		t.Fatal("engine does not implement MultiGroupedAggregateReader")
	}

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total_price"},
		{Fn: metaengine.AggregateSum, Column: "quantity", Alias: "total_qty"},
	}

	result, err := mgr.MultiGroupedAggregate(ctx, col, specs, "category", nil)
	if err != nil {
		t.Fatalf("MultiGroupedAggregate: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result))
	}

	groupMap := make(map[string]metaengine.GroupedAggregateRow, len(result))
	for _, row := range result {
		groupMap[row.Group] = row
	}

	elec := groupMap["electronics"]
	if elec.Values["count"] != 3 {
		t.Errorf("electronics count: got %v, want 3", elec.Values["count"])
	}

	if elec.Values["total_price"] != 1325.0 {
		t.Errorf("electronics total_price: got %v, want 1325", elec.Values["total_price"])
	}

	if elec.Values["total_qty"] != 40 { // 5+20+15
		t.Errorf("electronics total_qty: got %v, want 40", elec.Values["total_qty"])
	}

	food := groupMap["food"]
	if food.Values["total_price"] != 4.5 {
		t.Errorf("food total_price: got %v, want 4.5", food.Values["total_price"])
	}
}

// --- DistinctReader (SELECT DISTINCT) ---

func TestDuckDB_Distinct(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_distinct"
	seedProducts(t, eng, col)

	ctx := context.Background()
	dr, ok := eng.(metaengine.DistinctReader)
	if !ok {
		t.Fatal("engine does not implement DistinctReader")
	}

	values, err := dr.DistinctValues(ctx, col, "category", nil)
	if err != nil {
		t.Fatalf("DistinctValues: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 distinct categories, got %d: %v", len(values), values)
	}

	categories := make(map[string]bool, len(values))
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string category, got %T: %v", v, v)
		}

		categories[s] = true
	}

	for _, expected := range []string{"electronics", "books", "food"} {
		if !categories[expected] {
			t.Errorf("missing category %q in distinct values: %v", expected, values)
		}
	}
}

// --- Planned-table path tests (with layout) ---

func TestDuckDB_Aggregate_Planned(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	const col = "products_planned"

	lp, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanApplier")
	}

	// Use ApplyLayoutPlan with explicit column types — inferColumnType maps
	// "price" to INTEGER (truncates decimals), so we construct a plan with
	// DOUBLE for price.
	plan := metaengine.LayoutPlan{
		Collection: col,
		Table:      "meta_planned_" + col,
		Columns: []metaengine.PlannedColumn{
			{Name: "category", Type: "VARCHAR"},
			{Name: "price", Type: "DOUBLE"},
			{Name: "quantity", Type: "INTEGER"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_category", Columns: []string{"category"}},
		},
	}

	if err := lp.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	seedProducts(t, eng, col)

	ctx := context.Background()
	ar := eng.(metaengine.AggregateReader)

	n, err := ar.Aggregate(ctx, col, metaengine.AggregateCount, "", nil)
	if err != nil {
		t.Fatalf("Planned COUNT: %v", err)
	}

	if n != 7 {
		t.Errorf("Planned count: got %v, want 7", n)
	}

	sum, err := ar.Aggregate(ctx, col, metaengine.AggregateSum, "price", nil)
	if err != nil {
		t.Fatalf("Planned SUM: %v", err)
	}

	if sum != 1376.5 {
		t.Errorf("Planned sum price: got %v, want 1376.5", sum)
	}

	// Planned grouped
	gr := eng.(metaengine.GroupedAggregateReader)
	result, err := gr.GroupedAggregate(ctx, col, metaengine.AggregateCount, "", "category", nil)
	if err != nil {
		t.Fatalf("Planned GroupedAggregate: %v", err)
	}

	if result["electronics"] != 3 {
		t.Errorf("Planned electronics count: got %v, want 3", result["electronics"])
	}

	// Planned multi
	mr := eng.(metaengine.MultiAggregateReader)
	multi, err := mr.MultiAggregate(ctx, col, []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "total"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "revenue"},
	}, nil)
	if err != nil {
		t.Fatalf("Planned MultiAggregate: %v", err)
	}

	if multi["total"] != 7 {
		t.Errorf("Planned multi total: got %v, want 7", multi["total"])
	}

	if multi["revenue"] != 1376.5 {
		t.Errorf("Planned multi revenue: got %v, want 1376.5", multi["revenue"])
	}

	// Planned multi-grouped
	mgr := eng.(metaengine.MultiGroupedAggregateReader)
	mgResult, err := mgr.MultiGroupedAggregate(ctx, col,
		[]metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "count"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total_price"},
		}, "category", nil)
	if err != nil {
		t.Fatalf("Planned MultiGroupedAggregate: %v", err)
	}

	if len(mgResult) != 3 {
		t.Fatalf("Planned multi-grouped: expected 3 groups, got %d", len(mgResult))
	}

	// Planned distinct
	dr := eng.(metaengine.DistinctReader)
	dv, err := dr.DistinctValues(ctx, col, "category", nil)
	if err != nil {
		t.Fatalf("Planned DistinctValues: %v", err)
	}

	if len(dv) != 3 {
		t.Errorf("Planned distinct: expected 3 categories, got %d", len(dv))
	}
}

func TestDuckDB_Aggregate_EmptyCollection(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()
	ar := eng.(metaengine.AggregateReader)

	n, err := ar.Aggregate(ctx, "empty", metaengine.AggregateCount, "", nil)
	if err != nil {
		t.Fatalf("Empty COUNT: %v", err)
	}

	if n != 0 {
		t.Errorf("Empty count: got %v, want 0", n)
	}

	gr := eng.(metaengine.GroupedAggregateReader)
	result, err := gr.GroupedAggregate(ctx, "empty", metaengine.AggregateCount, "", "category", nil)
	if err != nil {
		t.Fatalf("Empty GroupedAggregate: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Empty grouped: expected 0 groups, got %d", len(result))
	}

	mr := eng.(metaengine.MultiAggregateReader)
	multi, err := mr.MultiAggregate(ctx, "empty", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
	}, nil)
	if err != nil {
		t.Fatalf("Empty MultiAggregate: %v", err)
	}

	if multi["count"] != 0 {
		t.Errorf("Empty multi count: got %v, want 0", multi["count"])
	}
}
