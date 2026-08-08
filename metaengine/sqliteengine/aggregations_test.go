package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
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

func TestSQLite_Aggregate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	seedAggData(t, ctx, eng)

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

func TestSQLite_GroupedAggregate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	seedAggData(t, ctx, eng)

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
		got, err := gr.GroupedAggregate(ctx, "items", metaengine.AggregateSum, "price", "status", nil)
		if err != nil {
			t.Fatalf("GroupedAggregate Sum: %v", err)
		}
		assertAggFloat(t, "GroupedSum[open]", got["open"], 60)
		assertAggFloat(t, "GroupedSum[closed]", got["closed"], -5)
	})

	t.Run("GroupedMin", func(t *testing.T) {
		got, err := gr.GroupedAggregate(ctx, "items", metaengine.AggregateMin, "price", "status", nil)
		if err != nil {
			t.Fatalf("GroupedAggregate Min: %v", err)
		}
		assertAggFloat(t, "GroupedMin[open]", got["open"], 10)
		assertAggFloat(t, "GroupedMin[closed]", got["closed"], -5)
	})

	t.Run("GroupedAvg", func(t *testing.T) {
		got, err := gr.GroupedAggregate(ctx, "items", metaengine.AggregateAvg, "price", "status", nil)
		if err != nil {
			t.Fatalf("GroupedAggregate Avg: %v", err)
		}
		assertAggFloat(t, "GroupedAvg[open]", got["open"], 20)
		assertAggFloat(t, "GroupedAvg[closed]", got["closed"], -2.5)
	})
}

func TestSQLite_MultiAggregate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	seedAggData(t, ctx, eng)

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

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	seedAggData(t, ctx, eng)

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

	ctx := context.Background()
	eng, cleanup := newAggSQLiteEngine(t)
	defer cleanup()

	seedAggData(t, ctx, eng)

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
