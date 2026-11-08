package adttest

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RunAggregateMatrix tests all 5 aggregate pushdown interfaces
// (AggregateReader, GroupedAggregateReader, MultiAggregateReader,
// MultiGroupedAggregateReader, ExplainableAggregate) across the given
// engines, verifying cross-engine result parity.
//
// Engines that do not implement a given interface are silently skipped
// for that subtest (same auto-skip behavior as RunMatrix).
//
// Usage from any engine module's tests:
//
//	func TestEngineAggregateMatrix(t *testing.T) {
//	    adttest.RunAggregateMatrix(t, []adttest.Factory{
//	        {Name: "sqlite", Create: func(t *testing.T) metaengine.Engine { return newSQLiteEngine(t) }},
//	        {Name: "duckdb", Create: func(t *testing.T) metaengine.Engine { return newDuckEngine(t) }},
//	    })
//	}
func RunAggregateMatrix(t *testing.T, factories []Factory) {
	t.Helper()

	ctx := context.Background()

	type engFix struct {
		name string
		eng  metaengine.Engine
	}

	engines := make([]engFix, 0, len(factories))

	for _, f := range factories {
		eng := f.Create(t)
		t.Cleanup(func() { _ = eng.Close() })
		seedAggregateData(t, ctx, eng)
		engines = append(engines, engFix{name: f.Name, eng: eng})
	}

	if len(engines) == 0 {
		t.Fatal("RunAggregateMatrix requires at least 1 engine")
	}

	t.Run("AggregateReader", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			label  string
			fn     metaengine.AggregateFn
			column string
			want   float64
		}{
			{"Count", metaengine.AggregateCount, "", 5},
			{"Sum", metaengine.AggregateSum, "price", 55},
			{"Min", metaengine.AggregateMin, "price", -5},
			{"Max", metaengine.AggregateMax, "price", 30},
			{"Avg", metaengine.AggregateAvg, "price", 11},
		}

		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				t.Parallel()

				for _, fx := range engines {
					ar, ok := fx.eng.(metaengine.AggregateReader)
					if !ok {
						t.Skipf("%s does not implement AggregateReader", fx.name)
					}

					got, err := ar.Aggregate(ctx, "items", tc.fn, tc.column, nil)
					if err != nil {
						t.Fatalf("%s.Aggregate(%s): %v", fx.name, tc.label, err)
					}

					if got != tc.want {
						t.Errorf("%s.Aggregate(%s) = %v, want %v", fx.name, tc.label, got, tc.want)
					}
				}
			})
		}
	})

	t.Run("GroupedAggregateReader", func(t *testing.T) {
		t.Parallel()

		for _, fx := range engines {
			gar, ok := fx.eng.(metaengine.GroupedAggregateReader)
			if !ok {
				continue
			}

			got, err := gar.GroupedAggregate(
				ctx,
				"items",
				metaengine.AggregateCount,
				"",
				"status",
				nil,
			)
			if err != nil {
				t.Fatalf("%s.GroupedAggregate: %v", fx.name, err)
			}

			if got["open"] != 3 {
				t.Errorf("%s GroupedCount/open = %v, want 3", fx.name, got["open"])
			}

			if got["closed"] != 2 {
				t.Errorf("%s GroupedCount/closed = %v, want 2", fx.name, got["closed"])
			}
		}
	})

	t.Run("MultiAggregateReader", func(t *testing.T) {
		t.Parallel()

		specs := []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		}

		for _, fx := range engines {
			mar, ok := fx.eng.(metaengine.MultiAggregateReader)
			if !ok {
				continue
			}

			got, err := mar.MultiAggregate(ctx, "items", specs, nil)
			if err != nil {
				t.Fatalf("%s.MultiAggregate: %v", fx.name, err)
			}

			if got["cnt"] != 5 {
				t.Errorf("%s MultiAggregate/cnt = %v, want 5", fx.name, got["cnt"])
			}

			if got["total"] != 55 {
				t.Errorf("%s MultiAggregate/total = %v, want 55", fx.name, got["total"])
			}
		}
	})

	t.Run("MultiGroupedAggregateReader", func(t *testing.T) {
		t.Parallel()

		specs := []metaengine.AggregateSpec{
			{Fn: metaengine.AggregateCount, Alias: "cnt"},
			{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		}

		for _, fx := range engines {
			mgar, ok := fx.eng.(metaengine.MultiGroupedAggregateReader)
			if !ok {
				continue
			}

			rows, err := mgar.MultiGroupedAggregate(ctx, "items", specs, "status", nil)
			if err != nil {
				t.Fatalf("%s.MultiGroupedAggregate: %v", fx.name, err)
			}

			sort.Slice(rows, func(i, j int) bool { return rows[i].Group < rows[j].Group })

			if len(rows) != 2 {
				t.Fatalf("%s MultiGroupedAggregate: %d rows, want 2", fx.name, len(rows))
			}

			if rows[0].Group != "closed" || rows[0].Values["cnt"] != 2 ||
				rows[0].Values["total"] != -5 {
				t.Errorf("%s closed row = %+v", fx.name, rows[0])
			}

			if rows[1].Group != "open" || rows[1].Values["cnt"] != 3 ||
				rows[1].Values["total"] != 60 {
				t.Errorf("%s open row = %+v", fx.name, rows[1])
			}
		}
	})

	t.Run("ExplainableAggregate", func(t *testing.T) {
		t.Parallel()

		for _, fx := range engines {
			ea, ok := fx.eng.(metaengine.ExplainableAggregate)
			if !ok {
				continue
			}

			sql, _ := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{
				Fn:     metaengine.AggregateCount,
				Column: "",
			})

			if sql == "" {
				t.Errorf("%s ExplainAggregateQuery returned empty SQL", fx.name)
			}
		}
	})

	// HostileFnRejected pins the aggregate-fn whitelist: AggregateFn is a
	// plain string alias interpolated into engine SQL verbatim, so an
	// unconstrained value is a SQL-injection vector. Every SQL engine must
	// reject it with a classified error (metaengine.invalid_aggregate_fn,
	// errors.Is metaengine.ErrInvalidAggregateFn) instead of executing.
	t.Run("HostileFnRejected", func(t *testing.T) {
		t.Parallel()

		hostile := metaengine.AggregateFn("COUNT(*) FROM meta_map; DROP TABLE meta_map; --")

		for _, fx := range engines {
			ar, ok := fx.eng.(metaengine.AggregateReader)
			if !ok {
				t.Skipf("%s does not implement AggregateReader", fx.name)
			}

			got, err := ar.Aggregate(ctx, "items", hostile, "price", nil)
			if err == nil {
				t.Fatalf("%s.Aggregate EXECUTED a hostile aggregate fn (returned %v)", fx.name, got)
			}

			if !errors.Is(err, metaengine.ErrInvalidAggregateFn) {
				t.Errorf("%s.Aggregate hostile-fn error does not wrap metaengine.ErrInvalidAggregateFn: %v", fx.name, err)
			}

			if mar, ok := fx.eng.(metaengine.MultiAggregateReader); ok {
				if _, err := mar.MultiAggregate(ctx, "items",
					[]metaengine.AggregateSpec{{Fn: hostile, Column: "price"}}, nil); err == nil {
					t.Fatalf("%s.MultiAggregate EXECUTED a hostile spec fn", fx.name)
				}
			}

			if ea, ok := fx.eng.(metaengine.ExplainableAggregate); ok {
				sql, _ := ea.ExplainAggregateQuery(ctx, "items", metaengine.ExplainAggregateOptions{Fn: hostile})
				if !strings.Contains(sql, "invalid aggregate fn") {
					t.Errorf("%s.ExplainAggregateQuery emitted hostile fn without the rejection marker: %s", fx.name, sql)
				}
			}
		}
	})
}

func seedAggregateData(t *testing.T, ctx context.Context, eng metaengine.Engine) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("engine does not implement MapBackend (required for aggregate test data seeding)")
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
