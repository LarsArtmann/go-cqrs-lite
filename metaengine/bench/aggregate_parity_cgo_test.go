//go:build cgo

package bench_test

import (
	"context"
	"sort"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestAggregateParity_DuckDB_vs_SQLite verifies that DuckDB and SQLite engines
// produce identical results for all aggregate interfaces given the same data.
// This is the cross-engine parity harness for aggregate pushdown.

type aggEngineFixture struct {
	name string
	eng  metaengine.Engine
}

func newAggEngines(t *testing.T) []aggEngineFixture {
	t.Helper()

	engines := []aggEngineFixture{}

	// SQLite
	sqliteEng, db := newSQLiteEngine()
	t.Cleanup(func() {
		_ = sqliteEng.Close()
		_ = db.Close()
	})
	engines = append(engines, aggEngineFixture{name: "sqlite", eng: sqliteEng})

	// DuckDB (CGo)
	duckEng := newDuckDBEngine(t)
	t.Cleanup(func() { _ = duckEng.Close() })
	engines = append(engines, aggEngineFixture{name: "duckdb", eng: duckEng})

	return engines
}

func seedParityAggData(t *testing.T, ctx context.Context, eng metaengine.Engine) {
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

func TestAggregateParity_DuckDB_vs_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engines := newAggEngines(t)

	for _, fx := range engines {
		seedParityAggData(t, ctx, fx.eng)
	}

	// Helper to run the same aggregate call on all engines and verify equality.
	runOnAll := func(t *testing.T, label string, fn func(metaengine.Engine) (float64, error)) {
		t.Helper()

		var first float64

		var firstEng string

		for i, fx := range engines {
			got, err := fn(fx.eng)
			if err != nil {
				t.Fatalf("%s: %s engine error: %v", label, fx.name, err)
			}

			if i == 0 {
				first = got
				firstEng = fx.name
			} else if got != first {
				t.Errorf("%s: %s=%v, %s=%v", label, firstEng, first, fx.name, got)
			}
		}
	}

	// Scalar aggregates
	runOnAll(t, "Count", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateCount, "", nil)
	})

	runOnAll(t, "Sum", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateSum, "price", nil)
	})

	runOnAll(t, "Min", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateMin, "price", nil)
	})

	runOnAll(t, "Max", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateMax, "price", nil)
	})

	runOnAll(t, "Avg", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateAvg, "price", nil)
	})

	// Grouped aggregates
	runOnAll(t, "GroupedCount/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateCount, "", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	runOnAll(t, "GroupedSum/closed", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateSum, "price", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["closed"], nil
	})

	runOnAll(t, "GroupedAvg/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateAvg, "price", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	// MultiAggregate
	runOnAll(t, "MultiAggregate/cnt", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.MultiAggregateReader).MultiAggregate(ctx, "items",
			[]metaengine.AggregateSpec{
				{Fn: metaengine.AggregateCount, Alias: "cnt"},
				{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			}, nil)
		if err != nil {
			return 0, err
		}

		return m["cnt"], nil
	})

	runOnAll(t, "MultiAggregate/total", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.MultiAggregateReader).MultiAggregate(ctx, "items",
			[]metaengine.AggregateSpec{
				{Fn: metaengine.AggregateCount, Alias: "cnt"},
				{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			}, nil)
		if err != nil {
			return 0, err
		}

		return m["total"], nil
	})

	// MultiGroupedAggregate — compare row count and per-group values
	t.Run("MultiGroupedAggregate", func(t *testing.T) {
		type result struct {
			group  string
			values map[string]float64
		}

		var prev []result

		for _, fx := range engines {
			rows, err := fx.eng.(metaengine.MultiGroupedAggregateReader).MultiGroupedAggregate(
				ctx, "items",
				[]metaengine.AggregateSpec{
					{Fn: metaengine.AggregateCount, Alias: "cnt"},
					{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
					{Fn: metaengine.AggregateAvg, Column: "price", Alias: "avg"},
				}, "status", nil)
			if err != nil {
				t.Fatalf("%s: MultiGroupedAggregate: %v", fx.name, err)
			}

			got := make([]result, len(rows))
			for i, r := range rows {
				got[i] = result{group: r.Group, values: r.Values}
			}

			sort.Slice(got, func(i, j int) bool { return got[i].group < got[j].group })

			if prev == nil {
				prev = got
			} else {
				if len(prev) != len(got) {
					t.Errorf("row count mismatch: %d vs %d", len(prev), len(got))
				}

				for i := range prev {
					if prev[i].group != got[i].group {
						t.Errorf("group mismatch at %d: %q vs %q", i, prev[i].group, got[i].group)
					}

					for k, v := range prev[i].values {
						if got[i].values[k] != v {
							t.Errorf("group %s key %s: %v vs %v",
								prev[i].group, k, v, got[i].values[k])
						}
					}
				}
			}
		}
	})

	// DistinctValues — compare count and sorted values
	t.Run("DistinctValues", func(t *testing.T) {
		var prevCount int

		for _, fx := range engines {
			vals, err := fx.eng.(metaengine.DistinctReader).DistinctValues(
				ctx, "items", "status", nil)
			if err != nil {
				t.Fatalf("%s: DistinctValues: %v", fx.name, err)
			}

			if prevCount == 0 {
				prevCount = len(vals)
			} else if prevCount != len(vals) {
				t.Errorf("distinct count mismatch: %d vs %d", prevCount, len(vals))
			}
		}
	})
}

// TestAggregateParity_WithFilters verifies that filtered aggregates produce
// identical results across DuckDB and SQLite engines despite different filter
// type-coercion strategies (DuckDB CAST AS DOUBLE, SQLite native types).
func TestAggregateParity_WithFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engines := newAggEngines(t)

	for _, fx := range engines {
		seedParityAggData(t, ctx, fx.eng)
	}

	statusOpenFilter := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "open"},
	}

	priceGteFilter := []metaengine.FilterSpec{
		{Column: "price", Op: metaengine.FilterGe, Value: float64(10)},
	}

	runOnAll := func(t *testing.T, label string, fn func(metaengine.Engine) (float64, error)) {
		t.Helper()

		var first float64

		var firstEng string

		for i, fx := range engines {
			got, err := fn(fx.eng)
			if err != nil {
				t.Fatalf("%s: %s engine error: %v", label, fx.name, err)
			}

			if i == 0 {
				first = got
				firstEng = fx.name
			} else if got != first {
				t.Errorf("%s: %s=%v, %s=%v", label, firstEng, first, fx.name, got)
			}
		}
	}

	// Scalar aggregates with status=open filter
	runOnAll(t, "FilteredCount/open", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateCount, "", statusOpenFilter)
	})

	runOnAll(t, "FilteredSum/open", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateSum, "price", statusOpenFilter)
	})

	runOnAll(t, "FilteredMin/open", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateMin, "price", statusOpenFilter)
	})

	runOnAll(t, "FilteredMax/open", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateMax, "price", statusOpenFilter)
	})

	runOnAll(t, "FilteredAvg/open", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateAvg, "price", statusOpenFilter)
	})

	// Numeric comparison filter: price >= 10
	runOnAll(t, "FilteredCount/price_ge_10", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateCount, "", priceGteFilter)
	})

	runOnAll(t, "FilteredSum/price_ge_10", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateSum, "price", priceGteFilter)
	})

	// Grouped aggregate with filter
	runOnAll(t, "FilteredGroupedCount/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateCount, "", "status", statusOpenFilter)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	// MultiAggregate with filter
	runOnAll(t, "FilteredMulti/total_open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.MultiAggregateReader).MultiAggregate(ctx, "items",
			[]metaengine.AggregateSpec{
				{Fn: metaengine.AggregateCount, Alias: "cnt"},
				{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
			}, statusOpenFilter)
		if err != nil {
			return 0, err
		}

		return m["total"], nil
	})

	// DistinctValues with filter
	t.Run("FilteredDistinct", func(t *testing.T) {
		var prevCount int

		for _, fx := range engines {
			vals, err := fx.eng.(metaengine.DistinctReader).DistinctValues(
				ctx, "items", "status", statusOpenFilter)
			if err != nil {
				t.Fatalf("%s: FilteredDistinctValues: %v", fx.name, err)
			}

			if prevCount == 0 {
				prevCount = len(vals)
			} else if prevCount != len(vals) {
				t.Errorf("filtered distinct count mismatch: %d vs %d", prevCount, len(vals))
			}
		}
	})
}
