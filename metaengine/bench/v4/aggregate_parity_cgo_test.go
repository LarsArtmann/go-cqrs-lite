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

	engines := make([]aggEngineFixture, 0, 2)

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

// runAggParityOnAll runs the same aggregate call on every engine fixture
// and verifies they all agree on the result.
func runAggParityOnAll(
	t *testing.T,
	engines []aggEngineFixture,
	label string,
	fn func(metaengine.Engine) (float64, error),
) {
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

// runScalarAggParity verifies the five unfiltered scalar aggregates agree
// across engines, with a label prefix separating the standard and
// planned-table harnesses.
func runScalarAggParity(t *testing.T, ctx context.Context, engines []aggEngineFixture, labelPrefix string) {
	t.Helper()

	scalars := []struct {
		label  string
		fn     metaengine.AggregateFn
		column string
	}{
		{"Count", metaengine.AggregateCount, ""},
		{"Sum", metaengine.AggregateSum, "price"},
		{"Min", metaengine.AggregateMin, "price"},
		{"Max", metaengine.AggregateMax, "price"},
		{"Avg", metaengine.AggregateAvg, "price"},
	}

	for _, agg := range scalars {
		fn, column := agg.fn, agg.column
		runAggParityOnAll(t, engines, labelPrefix+agg.label, func(e metaengine.Engine) (float64, error) {
			return e.(metaengine.AggregateReader).Aggregate(ctx, "items", fn, column, nil)
		})
	}
}

// runFilteredScalarAggParity verifies the five scalar aggregates agree
// across engines when pushed through one filter, labeled by its suffix
// (e.g. "open" for the status=open equality filter).
func runFilteredScalarAggParity(
	t *testing.T,
	ctx context.Context,
	engines []aggEngineFixture,
	filter []metaengine.FilterSpec,
	labelSuffix string,
) {
	t.Helper()

	scalars := []struct {
		label  string
		fn     metaengine.AggregateFn
		column string
	}{
		{"Count", metaengine.AggregateCount, ""},
		{"Sum", metaengine.AggregateSum, "price"},
		{"Min", metaengine.AggregateMin, "price"},
		{"Max", metaengine.AggregateMax, "price"},
		{"Avg", metaengine.AggregateAvg, "price"},
	}

	for _, agg := range scalars {
		fn, column := agg.fn, agg.column
		runAggParityOnAll(t, engines, "Filtered"+agg.label+"/"+labelSuffix,
			func(e metaengine.Engine) (float64, error) {
				return e.(metaengine.AggregateReader).Aggregate(ctx, "items", fn, column, filter)
			})
	}
}

func TestAggregateParity_DuckDB_vs_SQLite(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()
	engines := newAggEngines(t)

	for _, fx := range engines {
		seedParityAggData(t, ctx, fx.eng)
	}

	runScalarAggParity(t, ctx, engines, "")

	// Grouped aggregates
	runAggParityOnAll(t, engines, "GroupedCount/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateCount, "", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	runAggParityOnAll(t, engines, "GroupedSum/closed", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateSum, "price", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["closed"], nil
	})

	runAggParityOnAll(t, engines, "GroupedAvg/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateAvg, "price", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	// MultiAggregate
	runAggParityOnAll(t, engines, "MultiAggregate/cnt", func(e metaengine.Engine) (float64, error) {
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

	runAggParityOnAll(t, engines, "MultiAggregate/total", func(e metaengine.Engine) (float64, error) {
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
		assertMultiGroupedAggParity(t, ctx, engines)
	})

	// DistinctValues — compare count and sorted values
	t.Run("DistinctValues", func(t *testing.T) {
		assertDistinctValuesParity(t, ctx, engines, nil)
	})
}

// assertMultiGroupedAggParity compares MultiGroupedAggregate rows (group
// keys and per-group values) across every engine fixture.
func assertMultiGroupedAggParity(t *testing.T, ctx context.Context, engines []aggEngineFixture) {
	t.Helper()

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
}

// assertDistinctValuesParity compares the distinct-value count for the
// status column across every engine fixture, optionally through a filter.
func assertDistinctValuesParity(
	t *testing.T,
	ctx context.Context,
	engines []aggEngineFixture,
	filter []metaengine.FilterSpec,
) {
	t.Helper()

	var prevCount int

	for _, fx := range engines {
		vals, err := fx.eng.(metaengine.DistinctReader).DistinctValues(
			ctx, "items", "status", filter)
		if err != nil {
			t.Fatalf("%s: DistinctValues: %v", fx.name, err)
		}

		if prevCount == 0 {
			prevCount = len(vals)
		} else if prevCount != len(vals) {
			t.Errorf("distinct count mismatch: %d vs %d", prevCount, len(vals))
		}
	}
}

// TestAggregateParity_WithFilters verifies that filtered aggregates produce
// identical results across DuckDB and SQLite engines despite different filter
// type-coercion strategies (DuckDB CAST AS DOUBLE, SQLite native types).
func TestAggregateParity_WithFilters(t *testing.T) { //nolint:tparallel
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

	runFilteredScalarAggParity(t, ctx, engines, statusOpenFilter, "open")

	// Numeric comparison filter: price >= 10
	runAggParityOnAll(t, engines, "FilteredCount/price_ge_10", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateCount, "", priceGteFilter)
	})

	runAggParityOnAll(t, engines, "FilteredSum/price_ge_10", func(e metaengine.Engine) (float64, error) {
		return e.(metaengine.AggregateReader).Aggregate(ctx, "items",
			metaengine.AggregateSum, "price", priceGteFilter)
	})

	// Grouped aggregate with filter
	runAggParityOnAll(t, engines, "FilteredGroupedCount/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateCount, "", "status", statusOpenFilter)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	// MultiAggregate with filter
	runAggParityOnAll(t, engines, "FilteredMulti/total_open", func(e metaengine.Engine) (float64, error) {
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
		assertDistinctValuesParity(t, ctx, engines, statusOpenFilter)
	})
}

// TestAggregateParity_PlannedTable_DuckDB_vs_SQLite verifies that DuckDB and
// SQLite produce identical aggregate results when using planned tables
// (column-extracted layout). The existing TestAggregateParity_DuckDB_vs_SQLite
// tests the standard json_extract path; this test fills the gap for the
// planned-path, which uses native SQL columns instead of json_extract.
func TestAggregateParity_PlannedTable_DuckDB_vs_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := metaengine.LayoutPlan{
		Collection: "items",
		Table:      "meta_planned_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "price", Type: "DOUBLE"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_planned_status", Columns: []string{"status"}},
		},
	}

	// SQLite: planned via constructor (NewPlannedSQLiteEngine).
	sqliteEng, db := newPlannedSQLiteEngine(t, []metaengine.LayoutPlan{plan})

	// DuckDB: planned via ApplyLayoutPlan (LayoutPlanApplier).
	duckEng := newDuckDBEngine(t)
	if lp, ok := duckEng.(metaengine.LayoutPlanApplier); ok {
		if err := lp.ApplyLayoutPlan(plan); err != nil {
			t.Fatalf("DuckDB ApplyLayoutPlan: %v", err)
		}
	} else {
		t.Fatal("DuckDB engine does not implement LayoutPlanApplier")
	}

	t.Cleanup(func() {
		_ = sqliteEng.Close()
		_ = db.Close()
		_ = duckEng.Close()
	})

	engines := []aggEngineFixture{
		{name: "sqlite", eng: sqliteEng},
		{name: "duckdb", eng: duckEng},
	}

	for _, fx := range engines {
		seedParityAggData(t, ctx, fx.eng)
	}

	runScalarAggParity(t, ctx, engines, "Planned")

	runAggParityOnAll(t, engines, "PlannedGroupedCount/open", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateCount, "", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["open"], nil
	})

	runAggParityOnAll(t, engines, "PlannedGroupedSum/closed", func(e metaengine.Engine) (float64, error) {
		m, err := e.(metaengine.GroupedAggregateReader).GroupedAggregate(ctx, "items",
			metaengine.AggregateSum, "price", "status", nil)
		if err != nil {
			return 0, err
		}

		return m["closed"], nil
	})
}
