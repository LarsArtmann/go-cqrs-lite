package mysqlengine_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQL_EmptyFilterInMatchesNoRows pins ruling S041-02: a FilterIn whose
// list is empty (or absent) matches NO rows, as in the memory engine, on the
// meta_map path and on a planned collection. The SQL builders used to drop
// the clause, so the filter widened the result to every row.
//
// art-dupl:accept cross-engine conformance test — separate go.mod per engine
func TestMySQL_EmptyFilterInMatchesNoRows(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	ctx := t.Context()
	mb := eng.(metaengine.MapBackend)
	ps := eng.(metaengine.PushdownScan)
	ex := eng.(metaengine.ExplainableScan)

	type emptyInRow struct {
		Status string  `json:"status"`
		N      float64 `json:"n"`
	}

	const (
		planless = "empty_in_map"
		planned  = "empty_in_planned"
	)

	collections := []string{planless}

	if lpa, ok := eng.(metaengine.LayoutPlanApplier); ok {
		if err := lpa.ApplyLayoutPlan(metaengine.BuildLayoutPlanFromType[emptyInRow](
			planned, []string{"status", "n"}, nil,
		)); err != nil {
			t.Fatalf("ApplyLayoutPlan: %v", err)
		}

		collections = append(collections, planned)
	}

	for _, col := range collections {
		for i := range 3 {
			if err := mb.MapSet(ctx, col, strconv.Itoa(i), emptyInRow{Status: "open", N: float64(i)}); err != nil {
				t.Fatalf("MapSet: %v", err)
			}
		}
	}

	for _, col := range collections {
		t.Run(col, func(t *testing.T) {
			control, err := ps.PushdownMapScan(ctx, col, nil, nil, nil, 10)
			if err != nil || len(control.Items) != 3 {
				t.Fatalf("control scan: items=%d err=%v", len(control.Items), err)
			}

			for _, list := range []any{[]any{}, nil} {
				in := metaengine.FilterSpec{Column: "n", Op: metaengine.FilterIn, Value: list}
				narrowing := []metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}, in}

				for _, filters := range [][]metaengine.FilterSpec{{in}, narrowing} {
					res, err := ps.PushdownMapScan(ctx, col, filters, nil, nil, 10)
					if err != nil {
						t.Fatalf("PushdownMapScan with empty IN list %v: %v", list, err)
					}

					if len(res.Items) != 0 {
						t.Errorf("empty FilterIn returned %d rows, want 0", len(res.Items))
					}
				}
			}

			sql, _ := ex.ExplainScanQuery(ctx, col, metaengine.ExplainOptions{Filters: []metaengine.FilterSpec{
				{Column: "n", Op: metaengine.FilterIn, Value: []any{}},
			}})
			if !strings.Contains(sql, "1 = 0") {
				t.Errorf("explain %q lacks the empty-IN predicate", sql)
			}
		})
	}
}
