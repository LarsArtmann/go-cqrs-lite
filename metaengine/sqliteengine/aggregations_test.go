package sqliteengine_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func newAggSQLiteEngine(t *testing.T) (metaengine.Engine, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng, func() {
		_ = eng.Close()
		_ = db.Close()
	}
}

// setupSeededAggTest creates an in-memory SQLite engine, seeds it with test
// data via seedAggData, and returns the context + engine. Cleanup is automatic.
func setupSeededAggTest(t *testing.T) (context.Context, metaengine.Engine) {
	t.Helper()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	t.Cleanup(cleanup)
	seedAggData(t, ctx, eng)

	return ctx, eng
}

func seedAggData(t *testing.T, ctx context.Context, eng metaengine.Engine) {
	t.Helper()

	mb := eng.(metaengine.MapBackend)

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

func TestSQLite_Aggregate(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	ar := eng.(metaengine.AggregateReader)

	t.Run("Count", func(t *testing.T) {
		n, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "", nil)
		if err != nil {
			t.Fatalf("Aggregate Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Count = %v, want 5", n)
		}
	})

	t.Run("Sum", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateSum, "price", nil)
		if err != nil {
			t.Fatalf("Aggregate Sum: %v", err)
		}
		assertAggFloat(t, "Sum", got, 55)
	})

	t.Run("Min_negative", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMin, "price", nil)
		if err != nil {
			t.Fatalf("Aggregate Min: %v", err)
		}
		assertAggFloat(t, "Min", got, -5)
	})

	t.Run("Max", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMax, "price", nil)
		if err != nil {
			t.Fatalf("Aggregate Max: %v", err)
		}
		assertAggFloat(t, "Max", got, 30)
	})

	t.Run("Avg", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateAvg, "price", nil)
		if err != nil {
			t.Fatalf("Aggregate Avg: %v", err)
		}
		assertAggFloat(t, "Avg", got, 11)
	})

	t.Run("Count_with_filter", func(t *testing.T) {
		n, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "",
			[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}})
		if err != nil {
			t.Fatalf("Aggregate Count filtered: %v", err)
		}
		if n != 3 {
			t.Errorf("Count(open) = %v, want 3", n)
		}
	})
}

func TestSQLite_GroupedAggregate(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	gr := eng.(metaengine.GroupedAggregateReader)

	t.Run("GroupedCount", func(t *testing.T) {
		got, err := gr.GroupedAggregate(ctx, "items", metaengine.AggregateCount, "", "status", nil)
		if err != nil {
			t.Fatalf("GroupedAggregate Count: %v", err)
		}
		if got["open"] != 3 {
			t.Errorf("GroupedCount[open] = %v, want 3", got["open"])
		}
		if got["closed"] != 2 {
			t.Errorf("GroupedCount[closed] = %v, want 2", got["closed"])
		}
	})

	t.Run("GroupedSum", func(t *testing.T) {
		got, err := gr.GroupedAggregate(
			ctx,
			"items",
			metaengine.AggregateSum,
			"price",
			"status",
			nil,
		)
		if err != nil {
			t.Fatalf("GroupedAggregate Sum: %v", err)
		}
		assertAggFloat(t, "GroupedSum[open]", got["open"], 60)
		assertAggFloat(t, "GroupedSum[closed]", got["closed"], -5)
	})

	t.Run("GroupedMin", func(t *testing.T) {
		got, err := gr.GroupedAggregate(
			ctx,
			"items",
			metaengine.AggregateMin,
			"price",
			"status",
			nil,
		)
		if err != nil {
			t.Fatalf("GroupedAggregate Min: %v", err)
		}
		assertAggFloat(t, "GroupedMin[open]", got["open"], 10)
		assertAggFloat(t, "GroupedMin[closed]", got["closed"], -5)
	})

	t.Run("GroupedAvg", func(t *testing.T) {
		got, err := gr.GroupedAggregate(
			ctx,
			"items",
			metaengine.AggregateAvg,
			"price",
			"status",
			nil,
		)
		if err != nil {
			t.Fatalf("GroupedAggregate Avg: %v", err)
		}
		assertAggFloat(t, "GroupedAvg[open]", got["open"], 20)
		assertAggFloat(t, "GroupedAvg[closed]", got["closed"], -2.5)
	})
}

func TestSQLite_MultiAggregate(t *testing.T) {
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	mr := eng.(metaengine.MultiAggregateReader)

	got, err := mr.MultiAggregate(ctx, "items", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "cnt"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateMin, Column: "price", Alias: "min_price"},
		{Fn: metaengine.AggregateMax, Column: "price", Alias: "max_price"},
	}, nil)
	if err != nil {
		t.Fatalf("MultiAggregate: %v", err)
	}

	if got["cnt"] != 5 {
		t.Errorf("cnt = %v, want 5", got["cnt"])
	}
	assertAggFloat(t, "total", got["total"], 55)
	assertAggFloat(t, "min_price", got["min_price"], -5)
	assertAggFloat(t, "max_price", got["max_price"], 30)
}

func TestSQLite_MultiGroupedAggregate(t *testing.T) {
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	mgr := eng.(metaengine.MultiGroupedAggregateReader)

	rows, err := mgr.MultiGroupedAggregate(ctx, "items", []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "cnt"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateAvg, Column: "price", Alias: "avg_price"},
	}, "status", nil)
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
	assertAggFloat(t, "open total", got["open"]["total"], 60)
	assertAggFloat(t, "open avg", got["open"]["avg_price"], 20)
	assertAggFloat(t, "closed avg", got["closed"]["avg_price"], -2.5)
}

func TestSQLite_DistinctValues(t *testing.T) {
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	dr := eng.(metaengine.DistinctReader)

	got, err := dr.DistinctValues(ctx, "items", "status", nil)
	if err != nil {
		t.Fatalf("DistinctValues: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("DistinctValues returned %d values, want 2", len(got))
	}
}

func assertAggFloat(t *testing.T, label string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}

func TestSQLite_ExplainAggregateQuery(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx, eng := setupSeededAggTest(t)

	ea := eng.(metaengine.ExplainableAggregate)

	t.Run("scalar", func(t *testing.T) {
		sql, args := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
			Fn:     metaengine.AggregateSum,
			Column: "price",
		})
		if !strings.Contains(sql, "SUM(json_extract") {
			t.Errorf("expected SUM(json_extract in SQL, got: %s", sql)
		}
		if len(args) != 1 || args[0] != "items" {
			t.Errorf("expected args=[items], got: %v", args)
		}
	})

	t.Run("grouped", func(t *testing.T) {
		sql, _ := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
			Fn:      metaengine.AggregateCount,
			GroupBy: "status",
		})
		if !strings.Contains(sql, "GROUP BY") {
			t.Errorf("expected GROUP BY in SQL, got: %s", sql)
		}
		if !strings.Contains(sql, "group_key") {
			t.Errorf("expected group_key in SQL, got: %s", sql)
		}
	})

	t.Run("multi", func(t *testing.T) {
		sql, _ := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
			Specs: []metaengine.AggregateSpec{
				{Fn: metaengine.AggregateCount, Alias: "cnt"},
				{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			},
		})
		if !strings.Contains(sql, "COUNT(*)") || !strings.Contains(sql, "SUM(json_extract") {
			t.Errorf("expected COUNT(*) and SUM in SQL, got: %s", sql)
		}
	})

	t.Run("distinct", func(t *testing.T) {
		sql, _ := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
			Distinct: "status",
		})
		if !strings.Contains(sql, "SELECT DISTINCT") {
			t.Errorf("expected SELECT DISTINCT in SQL, got: %s", sql)
		}
	})

	t.Run("with_filter", func(t *testing.T) {
		sql, args := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
			Fn: metaengine.AggregateCount,
			Filters: []metaengine.FilterSpec{
				{Column: "status", Op: metaengine.FilterEq, Value: "open"},
			},
		})
		if !strings.Contains(sql, "json_extract") || !strings.Contains(sql, "= ?") {
			t.Errorf("expected filter in SQL, got: %s", sql)
		}
		if len(args) != 2 {
			t.Errorf("expected 2 args (collection + filter), got %d: %v", len(args), args)
		}
	})
}

func TestSQLite_Aggregate_PlannedTable(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	plan := metaengine.LayoutPlan{
		Collection: "items_planned",
		Table:      "meta_planned_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "price", Type: "REAL"},
			{Name: "category", Type: "TEXT"},
		},
	}

	eng, err := sqliteengine.NewPlannedSQLiteEngine(db, []metaengine.LayoutPlan{plan})
	if err != nil {
		t.Fatalf("NewPlannedSQLiteEngine: %v", err)
	}
	defer eng.Close()

	seedAggData(t, ctx, eng)
	// Also seed planned collection by re-mapping to "items_planned"
	mb := eng.(metaengine.MapBackend)
	for _, item := range []struct {
		id, status string
		price      float64
	}{
		{"a", "open", 10},
		{"b", "open", 20},
		{"c", "closed", -5},
		{"d", "closed", 0},
		{"e", "open", 30},
	} {
		val := map[string]any{"id": item.id, "status": item.status, "price": item.price}
		if err := mb.MapSet(ctx, "items_planned", item.id, val); err != nil {
			t.Fatalf("MapSet planned %s: %v", item.id, err)
		}
	}

	ar := eng.(metaengine.AggregateReader)
	gr := eng.(metaengine.GroupedAggregateReader)
	mr := eng.(metaengine.MultiAggregateReader)
	mgr := eng.(metaengine.MultiGroupedAggregateReader)
	dr := eng.(metaengine.DistinctReader)

	t.Run("Count", func(t *testing.T) {
		n, err := ar.Aggregate(ctx, "items_planned", metaengine.AggregateCount, "", nil)
		if err != nil {
			t.Fatalf("Planned Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Planned Count = %v, want 5", n)
		}
	})

	t.Run("Sum_price", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items_planned", metaengine.AggregateSum, "price", nil)
		if err != nil {
			t.Fatalf("Planned Sum: %v", err)
		}
		assertAggFloat(t, "Planned Sum", got, 55)
	})

	t.Run("Min_negative", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items_planned", metaengine.AggregateMin, "price", nil)
		if err != nil {
			t.Fatalf("Planned Min: %v", err)
		}
		assertAggFloat(t, "Planned Min", got, -5)
	})

	t.Run("Max", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items_planned", metaengine.AggregateMax, "price", nil)
		if err != nil {
			t.Fatalf("Planned Max: %v", err)
		}
		assertAggFloat(t, "Planned Max", got, 30)
	})

	t.Run("Avg", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items_planned", metaengine.AggregateAvg, "price", nil)
		if err != nil {
			t.Fatalf("Planned Avg: %v", err)
		}
		assertAggFloat(t, "Planned Avg", got, 11)
	})

	t.Run("GroupedSum", func(t *testing.T) {
		got, err := gr.GroupedAggregate(
			ctx,
			"items_planned",
			metaengine.AggregateSum,
			"price",
			"status",
			nil,
		)
		if err != nil {
			t.Fatalf("Planned GroupedSum: %v", err)
		}
		assertAggFloat(t, "Planned GroupedSum[open]", got["open"], 60)
		assertAggFloat(t, "Planned GroupedSum[closed]", got["closed"], -5)
	})

	t.Run("MultiAggregate", func(t *testing.T) {
		got, err := mr.MultiAggregate(ctx, "items_planned", []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			{Fn: metaengine.AggregateMin, Column: "price", Alias: "min_price"},
		}, nil)
		if err != nil {
			t.Fatalf("Planned MultiAggregate: %v", err)
		}
		if got["cnt"] != 5 {
			t.Errorf("Planned cnt = %v, want 5", got["cnt"])
		}
		assertAggFloat(t, "Planned total", got["total"], 55)
		assertAggFloat(t, "Planned min_price", got["min_price"], -5)
	})

	t.Run("MultiGroupedAggregate", func(t *testing.T) {
		rows, err := mgr.MultiGroupedAggregate(ctx, "items_planned", []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		}, "status", nil)
		if err != nil {
			t.Fatalf("Planned MultiGroupedAggregate: %v", err)
		}
		got := make(map[string]map[string]float64, len(rows))
		for _, r := range rows {
			got[r.Group] = r.Values
		}
		if got["open"]["cnt"] != 3 {
			t.Errorf("Planned open cnt = %v, want 3", got["open"]["cnt"])
		}
		assertAggFloat(t, "Planned open total", got["open"]["total"], 60)
	})

	t.Run("DistinctValues", func(t *testing.T) {
		got, err := dr.DistinctValues(ctx, "items_planned", "status", nil)
		if err != nil {
			t.Fatalf("Planned DistinctValues: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Planned DistinctValues returned %d, want 2", len(got))
		}
	})

	t.Run("ExplainAggregateQuery_planned", func(t *testing.T) {
		ea := eng.(metaengine.ExplainableAggregate)
		sql, _ := ea.ExplainAggregateQuery(ctx, "items_planned", metaengine.ExplainAggregateOptions{
			Fn:     metaengine.AggregateSum,
			Column: "price",
		})
		// Planned path should use direct column refs, not json_extract
		if strings.Contains(sql, "json_extract") {
			t.Errorf("planned path should not use json_extract, got: %s", sql)
		}
		if !strings.Contains(sql, "SUM") {
			t.Errorf("expected SUM in SQL, got: %s", sql)
		}
		if !strings.Contains(sql, "meta_planned_items") {
			t.Errorf("expected planned table name in SQL, got: %s", sql)
		}
	})
}

func TestSQLite_Aggregate_EmptyCollection(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	// Don't seed any data — test edge cases on empty collection

	ar := eng.(metaengine.AggregateReader)
	gr := eng.(metaengine.GroupedAggregateReader)
	mr := eng.(metaengine.MultiAggregateReader)
	dr := eng.(metaengine.DistinctReader)

	t.Run("Count_empty", func(t *testing.T) {
		n, err := ar.Aggregate(ctx, "empty", metaengine.AggregateCount, "", nil)
		if err != nil {
			t.Fatalf("Aggregate Count on empty: %v", err)
		}
		if n != 0 {
			t.Errorf("Count on empty = %v, want 0", n)
		}
	})

	t.Run("Sum_empty", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "empty", metaengine.AggregateSum, "price", nil)
		if err != nil {
			t.Fatalf("Aggregate Sum on empty: %v", err)
		}
		if got != 0 {
			t.Errorf("Sum on empty = %v, want 0", got)
		}
	})

	t.Run("GroupedAggregate_empty", func(t *testing.T) {
		got, err := gr.GroupedAggregate(ctx, "empty", metaengine.AggregateCount, "", "status", nil)
		if err != nil {
			t.Fatalf("GroupedAggregate on empty: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GroupedAggregate on empty = %v entries, want 0", len(got))
		}
	})

	t.Run("MultiAggregate_empty", func(t *testing.T) {
		got, err := mr.MultiAggregate(ctx, "empty", []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		}, nil)
		if err != nil {
			t.Fatalf("MultiAggregate on empty: %v", err)
		}
		if got["cnt"] != 0 {
			t.Errorf("MultiAggregate cnt on empty = %v, want 0", got["cnt"])
		}
	})

	t.Run("DistinctValues_empty", func(t *testing.T) {
		got, err := dr.DistinctValues(ctx, "empty", "status", nil)
		if err != nil {
			t.Fatalf("DistinctValues on empty: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("DistinctValues on empty = %d, want 0", len(got))
		}
	})
}

func TestSQLite_Aggregate_NullValues(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	t.Cleanup(cleanup)

	mb := eng.(metaengine.MapBackend)

	// Insert items where some lack the "price" field (JSON null).
	items := []struct {
		id  string
		val map[string]any
	}{
		{"a", map[string]any{"id": "a", "status": "open", "price": 10.0}},
		{"b", map[string]any{"id": "b", "status": "open"}}, // no price
		{"c", map[string]any{"id": "c", "status": "closed", "price": 0.0}},
		{"d", map[string]any{"id": "d", "status": "closed"}}, // no price
		{"e", map[string]any{"id": "e", "status": "open", "price": 30.0}},
	}

	for _, item := range items {
		if err := mb.MapSet(ctx, "items", item.id, item.val); err != nil {
			t.Fatalf("MapSet %s: %v", item.id, err)
		}
	}

	ar := eng.(metaengine.AggregateReader)

	// COUNT counts ALL rows regardless of missing fields.
	t.Run("Count_all", func(t *testing.T) {
		n, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "", nil)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if n != 5 {
			t.Errorf("Count = %v, want 5", n)
		}
	})

	// SUM excludes rows with missing "price" (SQL NULL semantics).
	t.Run("Sum_excludes_null", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateSum, "price", nil)
		if err != nil {
			t.Fatalf("Sum: %v", err)
		}
		// 10 + 0 + 30 = 40 (b and d excluded)
		assertAggFloat(t, "Sum with nulls", got, 40)
	})

	// AVG averages only non-null prices: (10+0+30)/3 = 13.33...
	t.Run("Avg_excludes_null", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateAvg, "price", nil)
		if err != nil {
			t.Fatalf("Avg: %v", err)
		}
		assertAggFloat(t, "Avg with nulls", got, 40.0/3.0)
	})

	// MIN/MAX skip nulls.
	t.Run("Min_excludes_null", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMin, "price", nil)
		if err != nil {
			t.Fatalf("Min: %v", err)
		}
		assertAggFloat(t, "Min with nulls", got, 0)
	})

	t.Run("Max_excludes_null", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMax, "price", nil)
		if err != nil {
			t.Fatalf("Max: %v", err)
		}
		assertAggFloat(t, "Max with nulls", got, 30)
	})
}

func TestSQLite_Aggregate_LargeDataset(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	t.Cleanup(cleanup)

	mb := eng.(metaengine.MapBackend)

	const n = 10_000
	var expectedSum float64

	for i := 0; i < n; i++ {
		price := float64(i) + 1
		expectedSum += price
		val := map[string]any{"id": fmtID(i), "status": "open", "price": price}
		if err := mb.MapSet(ctx, "items", fmtID(i), val); err != nil {
			t.Fatalf("MapSet %d: %v", i, err)
		}
	}

	ar := eng.(metaengine.AggregateReader)

	t.Run("Count", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateCount, "", nil)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if got != float64(n) {
			t.Errorf("Count = %v, want %d", got, n)
		}
	})

	t.Run("Sum", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateSum, "price", nil)
		if err != nil {
			t.Fatalf("Sum: %v", err)
		}
		// Sum of 1..10000 = n*(n+1)/2
		assertAggFloat(t, "Sum 10K", got, float64(n)*float64(n+1)/2)
	})

	t.Run("Avg", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateAvg, "price", nil)
		if err != nil {
			t.Fatalf("Avg: %v", err)
		}
		assertAggFloat(t, "Avg 10K", got, expectedSum/float64(n))
	})

	t.Run("Min", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMin, "price", nil)
		if err != nil {
			t.Fatalf("Min: %v", err)
		}
		assertAggFloat(t, "Min 10K", got, 1)
	})

	t.Run("Max", func(t *testing.T) {
		got, err := ar.Aggregate(ctx, "items", metaengine.AggregateMax, "price", nil)
		if err != nil {
			t.Fatalf("Max: %v", err)
		}
		assertAggFloat(t, "Max 10K", got, float64(n))
	})
}

func fmtID(i int) string {
	return "item-" + itoa(i)
}

func itoa(i int) string {
	const maxInt = 1 << 31
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
