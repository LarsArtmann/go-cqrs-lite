package mysqlengine

import (
	"reflect"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// queryParityInputs exercise every clause PushdownMapScan emits on meta_map:
// an equality filter, an IN (...) list, a compound keyset cursor and a
// descending sort with the key tiebreak.
var queryParityFilters = []metaengine.FilterSpec{
	{Column: "status", Op: metaengine.FilterEq, Value: "open"},
	{Column: "n", Op: metaengine.FilterIn, Value: []any{float64(1), float64(2)}},
}

// TestExplainScanQueryIsThePushdownQuery pins S055-02: ExplainScanQuery
// returns exactly the SQL and args PushdownMapScan builds, on both server
// dialects. It used to render its own twin: no IN (...) form, no compound
// cursor predicate, no key tiebreak in ORDER BY.
func TestExplainScanQueryIsThePushdownQuery(t *testing.T) {
	t.Parallel()

	sort := &metaengine.SortSpec{Column: "priority", Desc: true}
	cursor := metaengine.SortKeyCursor{Sort: float64(2), Key: []byte("k1")}

	for _, dialect := range []string{dialectMySQL, dialectMariaDB} {
		e := &mysqlEngine{dialect: dialect}

		want, wantArgs, err := e.buildMapScanQuery("c", queryParityFilters, sort, cursor, 10)
		if err != nil {
			t.Fatalf("%s: buildMapScanQuery: %v", dialect, err)
		}

		for _, wantSubstr := range []string{" IN (", "`key` > ?", ", `key`"} {
			if !strings.Contains(want, wantSubstr) {
				t.Fatalf("%s: pushdown query %q lacks %q", dialect, want, wantSubstr)
			}
		}

		got, gotArgs := e.ExplainScanQuery(t.Context(), "c", metaengine.ExplainOptions{
			Filters: queryParityFilters, Sort: sort, Cursor: cursor, Limit: 10,
		})

		if got != want {
			t.Errorf("%s: ExplainScanQuery = %q, want the pushdown query %q", dialect, got, want)
		}

		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("%s: explain args = %v, want pushdown args %v", dialect, gotArgs, wantArgs)
		}
	}
}

// TestMapScanQueryRejectsInvalidQueryInput pins the mysqlengine side of the
// shared validation gate on the surfaces that need no server: a hostile
// operator, filter column or sort column yields an error from the scan
// builder and the rejected marker from both explain surfaces.
func TestMapScanQueryRejectsInvalidQueryInput(t *testing.T) {
	t.Parallel()

	hostile := []metaengine.FilterSpec{
		{Column: "status", Op: "= 1 OR 1 =", Value: "open"},
		{Column: "x') OR ('1'='1", Op: metaengine.FilterEq, Value: "open"},
	}

	for _, dialect := range []string{dialectMySQL, dialectMariaDB} {
		e := &mysqlEngine{dialect: dialect}

		for _, f := range hostile {
			filters := []metaengine.FilterSpec{f}

			if _, _, err := e.buildMapScanQuery("c", filters, nil, nil, 10); err == nil {
				t.Errorf("%s: hostile filter %+v was not rejected", dialect, f)
			}

			for name, sql := range map[string]string{
				"scan":       firstStr(e.ExplainScanQuery(t.Context(), "c", metaengine.ExplainOptions{Filters: filters})),
				"aggregate":  firstStr(e.ExplainAggregateQuery(t.Context(), "c", metaengine.ExplainAggregateOptions{Filters: filters})),
				"agg column": firstStr(e.ExplainAggregateQuery(t.Context(), "c", metaengine.ExplainAggregateOptions{Column: "n); DROP TABLE x; --"})),
				"agg group":  firstStr(e.ExplainAggregateQuery(t.Context(), "c", metaengine.ExplainAggregateOptions{GroupBy: "status'"})),
				"agg spec": firstStr(e.ExplainAggregateQuery(t.Context(), "c", metaengine.ExplainAggregateOptions{
					Specs: []metaengine.AggregateSpec{{Fn: metaengine.AggregateSum, Column: "n'--"}},
				})),
				"sort": firstStr(e.ExplainScanQuery(t.Context(), "c", metaengine.ExplainOptions{
					Sort: &metaengine.SortSpec{Column: "n); DROP TABLE x; --"},
				})),
			} {
				if sql != rejectedExplain {
					t.Errorf("%s: %s explain = %q, want the rejected marker", dialect, name, sql)
				}
			}
		}
	}
}

func firstStr(s string, _ []any) string { return s }
