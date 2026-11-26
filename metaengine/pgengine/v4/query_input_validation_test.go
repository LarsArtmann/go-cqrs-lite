package pgengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPg_QueryInputsValidated pins FIXER-pg-validate: every pgengine surface
// that splices caller-supplied filter or column text into SQL refuses a
// hostile operator, a hostile filter column and a hostile value / group-by /
// sort column, also when the engine is called directly (bypassing the typed
// reader's validation).
func TestPg_QueryInputsValidated(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)
	ctx := t.Context()

	agg := eng.(metaengine.AggregateReader)
	gagg := eng.(metaengine.GroupedAggregateReader)
	magg := eng.(metaengine.MultiAggregateReader)
	mgagg := eng.(metaengine.MultiGroupedAggregateReader)
	distinct := eng.(metaengine.DistinctReader)
	pushdown := eng.(metaengine.PushdownScan)
	exScan := eng.(metaengine.ExplainableScan)
	exAgg := eng.(metaengine.ExplainableAggregate)

	// rejected reports whether the surface refused the input.
	type probe struct {
		name string
		// col marks the surfaces that splice the value column; grp the
		// group-by / sort column.
		col, grp bool
		rejected func(filters []metaengine.FilterSpec, column, groupBy string) bool
	}

	probes := []probe{
		{name: "PushdownMapScan", grp: true, rejected: func(fs []metaengine.FilterSpec, _, grp string) bool {
			_, err := pushdown.PushdownMapScan(ctx, "validated_input", fs, &metaengine.SortSpec{Column: grp}, nil, 10)

			return err != nil
		}},
		{name: "Aggregate", col: true, rejected: func(fs []metaengine.FilterSpec, col, _ string) bool {
			_, err := agg.Aggregate(ctx, "validated_input", metaengine.AggregateSum, col, fs)

			return err != nil
		}},
		{
			name: "GroupedAggregate",
			col:  true,
			grp:  true,
			rejected: func(fs []metaengine.FilterSpec, col, grp string) bool {
				_, err := gagg.GroupedAggregate(ctx, "validated_input", metaengine.AggregateSum, col, grp, fs)

				return err != nil
			},
		},
		{name: "MultiAggregate", col: true, rejected: func(fs []metaengine.FilterSpec, col, _ string) bool {
			_, err := magg.MultiAggregate(ctx, "validated_input",
				[]metaengine.AggregateSpec{{Fn: metaengine.AggregateSum, Column: col}}, fs)

			return err != nil
		}},
		{
			name: "MultiGroupedAggregate",
			col:  true,
			grp:  true,
			rejected: func(fs []metaengine.FilterSpec, col, grp string) bool {
				_, err := mgagg.MultiGroupedAggregate(ctx, "validated_input",
					[]metaengine.AggregateSpec{{Fn: metaengine.AggregateSum, Column: col}}, grp, fs)

				return err != nil
			},
		},
		{name: "DistinctValues", col: true, rejected: func(fs []metaengine.FilterSpec, col, _ string) bool {
			_, err := distinct.DistinctValues(ctx, "validated_input", col, fs)

			return err != nil
		}},
		{name: "ExplainScanQuery", grp: true, rejected: func(fs []metaengine.FilterSpec, _, grp string) bool {
			sql, _ := exScan.ExplainScanQuery(ctx, "validated_input",
				metaengine.ExplainOptions{Filters: fs, Sort: &metaengine.SortSpec{Column: grp}})

			return strings.HasPrefix(sql, "/* rejected")
		}},
		{
			name: "ExplainAggregateQuery",
			col:  true,
			grp:  true,
			rejected: func(fs []metaengine.FilterSpec, col, grp string) bool {
				sql, _ := exAgg.ExplainAggregateQuery(ctx, "validated_input", metaengine.ExplainAggregateOptions{
					Fn: metaengine.AggregateSum, Column: col, GroupBy: grp, Filters: fs,
				})

				return strings.HasPrefix(sql, "/* rejected")
			},
		},
	}

	const (
		goodColumn = "amount"
		goodGroup  = "status"
	)

	hostileFilters := [][]metaengine.FilterSpec{
		{{Column: "status", Op: "= 1 OR 1 =", Value: "open"}},
		{{Column: "x') OR ('1'='1", Op: metaengine.FilterEq, Value: "open"}},
	}
	hostileColumns := []string{"n); DROP TABLE x; --", "status'"}

	for _, p := range probes {
		if p.rejected(nil, goodColumn, goodGroup) {
			t.Fatalf("%s: clean input was rejected", p.name)
		}

		for _, filters := range hostileFilters {
			if !p.rejected(filters, goodColumn, goodGroup) {
				t.Errorf("%s: hostile filter %+v was not rejected", p.name, filters)
			}
		}

		for _, hostile := range hostileColumns {
			if p.col && !p.rejected(nil, hostile, goodGroup) {
				t.Errorf("%s: hostile value column %q was not rejected", p.name, hostile)
			}

			if p.grp && !p.rejected(nil, goodColumn, hostile) {
				t.Errorf("%s: hostile group-by/sort column %q was not rejected", p.name, hostile)
			}
		}
	}
}
