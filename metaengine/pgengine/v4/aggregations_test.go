package pgengine_test

import (
	"context"
	"strings"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// aggregations_test.go tests all five Postgres aggregate pushdown interfaces
// (AggregateReader, GroupedAggregateReader, MultiAggregateReader,
// MultiGroupedAggregateReader, DistinctReader) plus ExplainableAggregate.
//
// DuckDB and SQLite have equivalent tests; this fills the gap where PG had
// only compile-time assertions (line 344 of aggregations.go) but zero
// functional tests.

func seedPGAggData(t *testing.T, ctx context.Context, eng metaengine.Engine) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	items := []struct {
		id     string
		status string
		price  float64
	}{
		{"a", "open", 10},
		{"b", "open", 20},
		{"c", "closed", -5},
		{"d", "closed", 0},
		{"e", "open", 30},
	}

	for _, item := range items {
		val := map[string]any{"id": item.id, "status": item.status, "price": item.price}
		if err := mb.MapSet(ctx, "items", item.id, val); err != nil {
			t.Fatalf("MapSet %s: %v", item.id, err)
		}
	}
}

func pgAggEngine(t *testing.T) (metaengine.Engine, func()) {
	t.Helper()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	return eng, func() { _ = eng.Close() }
}

func assertPGAggFloat(t *testing.T, label string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}

func TestPostgres_Aggregate(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	seedPGAggData(t, ctx, eng)
	ar := eng.(metaengine.AggregateReader)

	count, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "", nil)
	if err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	assertPGAggFloat(t, "COUNT", count, 5)

	sum, err := ar.Aggregate(ctx, "items", metaengine.AggregateSum, "price", nil)
	if err != nil {
		t.Fatalf("SUM(price): %v", err)
	}
	assertPGAggFloat(t, "SUM(price)", sum, 55)

	minVal, err := ar.Aggregate(ctx, "items", metaengine.AggregateMin, "price", nil)
	if err != nil {
		t.Fatalf("MIN(price): %v", err)
	}
	assertPGAggFloat(t, "MIN(price)", minVal, -5)

	maxVal, err := ar.Aggregate(ctx, "items", metaengine.AggregateMax, "price", nil)
	if err != nil {
		t.Fatalf("MAX(price): %v", err)
	}
	assertPGAggFloat(t, "MAX(price)", maxVal, 30)

	avg, err := ar.Aggregate(ctx, "items", metaengine.AggregateAvg, "price", nil)
	if err != nil {
		t.Fatalf("AVG(price): %v", err)
	}
	assertPGAggFloat(t, "AVG(price)", avg, 11)

	filtered, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}})
	if err != nil {
		t.Fatalf("COUNT filtered: %v", err)
	}
	assertPGAggFloat(t, "COUNT(open)", filtered, 3)
}

func TestPostgres_GroupedAggregate(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	seedPGAggData(t, ctx, eng)
	gr := eng.(metaengine.GroupedAggregateReader)

	grouped, err := gr.GroupedAggregate(ctx, "items", metaengine.AggregateCount, "", "status", nil)
	if err != nil {
		t.Fatalf("GroupedAggregate COUNT: %v", err)
	}
	if len(grouped) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(grouped))
	}
	assertPGAggFloat(t, "open count", grouped["open"], 3)
	assertPGAggFloat(t, "closed count", grouped["closed"], 2)

	groupedSum, err := gr.GroupedAggregate(
		ctx,
		"items",
		metaengine.AggregateSum,
		"price",
		"status",
		nil,
	)
	if err != nil {
		t.Fatalf("GroupedAggregate SUM: %v", err)
	}
	assertPGAggFloat(t, "open sum", groupedSum["open"], 60)
	assertPGAggFloat(t, "closed sum", groupedSum["closed"], -5)
}

func TestPostgres_MultiAggregate(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	seedPGAggData(t, ctx, eng)
	mr := eng.(metaengine.MultiAggregateReader)

	multi, err := mr.MultiAggregate(ctx, "items", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateMin, Column: "price", Alias: "min_price"},
		{Fn: metaengine.AggregateMax, Column: "price", Alias: "max_price"},
	}, nil)
	if err != nil {
		t.Fatalf("MultiAggregate: %v", err)
	}
	assertPGAggFloat(t, "count", multi["count"], 5)
	assertPGAggFloat(t, "total", multi["total"], 55)
	assertPGAggFloat(t, "min_price", multi["min_price"], -5)
	assertPGAggFloat(t, "max_price", multi["max_price"], 30)
}

func TestPostgres_MultiGroupedAggregate(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	seedPGAggData(t, ctx, eng)
	mgr := eng.(metaengine.MultiGroupedAggregateReader)

	rows, err := mgr.MultiGroupedAggregate(ctx, "items", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateAvg, Column: "price", Alias: "avg_price"},
	}, "status", nil)
	if err != nil {
		t.Fatalf("MultiGroupedAggregate: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(rows))
	}

	want := map[string]struct {
		count    float64
		total    float64
		avgPrice float64
	}{
		"open":   {3, 60, 20},
		"closed": {2, -5, -2.5},
	}

	for _, row := range rows {
		exp, ok := want[row.Group]
		if !ok {
			t.Errorf("unexpected group %q", row.Group)
			continue
		}

		assertPGAggFloat(t, row.Group+".count", row.Values["count"], exp.count)
		assertPGAggFloat(t, row.Group+".total", row.Values["total"], exp.total)
		assertPGAggFloat(t, row.Group+".avg_price", row.Values["avg_price"], exp.avgPrice)
	}
}

func TestPostgres_DistinctValues(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	seedPGAggData(t, ctx, eng)
	dr := eng.(metaengine.DistinctReader)

	vals, err := dr.DistinctValues(ctx, "items", "status", nil)
	if err != nil {
		t.Fatalf("DistinctValues: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("expected 2 distinct values, got %d: %v", len(vals), vals)
	}

	want := map[string]bool{"open": false, "closed": false}
	for _, v := range vals {
		s, ok := v.(string)
		if !ok {
			t.Errorf("distinct value %v is %T, not string", v, v)
			continue
		}

		if _, exists := want[s]; !exists {
			t.Errorf("unexpected distinct value %q", s)
		} else {
			want[s] = true
		}
	}

	for val, found := range want {
		if !found {
			t.Errorf("expected distinct value %q not returned", val)
		}
	}
}

func TestPostgres_Aggregate_EmptyCollection(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	ar := eng.(metaengine.AggregateReader)

	count, err := ar.Aggregate(ctx, "nonexistent", metaengine.AggregateCount, "", nil)
	if err != nil {
		t.Fatalf("Empty COUNT: %v", err)
	}
	assertPGAggFloat(t, "Empty count", count, 0)

	mr := eng.(metaengine.MultiAggregateReader)
	multi, err := mr.MultiAggregate(ctx, "nonexistent", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
	}, nil)
	if err != nil {
		t.Fatalf("Empty MultiAggregate: %v", err)
	}
	assertPGAggFloat(t, "Empty multi count", multi["count"], 0)
}

func TestPostgres_ExplainAggregateQuery(t *testing.T) {
	t.Parallel()

	eng, cleanup := pgAggEngine(t)
	defer cleanup()

	ctx := context.Background()
	ea, ok := eng.(metaengine.ExplainableAggregate)
	if !ok {
		t.Fatal("engine does not implement ExplainableAggregate")
	}

	sqlStr, args := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
		Fn:     metaengine.AggregateSum,
		Column: "price",
	})
	if sqlStr == "" {
		t.Error("expected non-empty SQL from ExplainAggregateQuery")
	}

	if !strings.Contains(sqlStr, "SUM") {
		t.Errorf("expected SUM keyword in SQL, got: %s", sqlStr)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 placeholder in SQL, got: %s", sqlStr)
	}

	if len(args) < 1 || args[0] != "items" {
		t.Errorf("expected first arg to be collection name %q, got: %v", "items", args)
	}
}
