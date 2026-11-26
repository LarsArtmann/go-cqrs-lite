package sqliteengine_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// hostileQueryInputs are operator and column texts that must never reach
// statement text on the aggregate / distinct / raw-scan / explain surfaces.
var hostileQueryInputs = []metaengine.FilterSpec{
	{Column: "status", Op: "= 1 OR 1 =", Value: "open"},
	{Column: "x') OR ('1'='1", Op: metaengine.FilterEq, Value: "open"},
}

var hostileColumns = []string{"n); DROP TABLE x; --", "status'"}

type surfaceProbe struct {
	name string
	// col / grp mark the surfaces that splice the value column / the
	// group-by column; the others are not exercised by those inputs.
	col, grp bool
	rejected bool
	run      func() bool
}

func (p surfaceProbe) probe(eng metaengine.Engine) bool { return false }

// surfaceScope selects the probes a test case applies to.
type surfaceScope struct{ col, grp bool }

func (s surfaceScope) matches(p surfaceProbe) bool {
	return (!p.col || s.col) && (!p.grp || s.grp)
}

var (
	allSurfaces = surfaceScope{col: true, grp: true}
	columnOnly  = surfaceScope{col: true}
	groupByOnly = surfaceScope{grp: true}
)

// probeQuerySurfaces calls every surface that splices column and filter
// text into SQL and reports, per surface, whether it refused the input.
func probeQuerySurfaces(
	eng metaengine.Engine,
	ctx context.Context,
	column, groupBy string,
	filters []metaengine.FilterSpec,
) []surfaceProbe {
	specs := []metaengine.AggregateSpec{{Fn: metaengine.AggregateSum, Column: column}}

	probes := []surfaceProbe{
		{name: "Aggregate", col: true, rejected: func() bool {
			_, err := eng.(metaengine.AggregateReader).Aggregate(ctx, "c", metaengine.AggregateSum, column, filters)
			return err != nil
		}()},
		{name: "GroupedAggregate", col: true, grp: true, rejected: func() bool {
			_, err := eng.(metaengine.GroupedAggregateReader).GroupedAggregate(
				ctx,
				"c",
				metaengine.AggregateSum,
				column,
				groupBy,
				filters,
			)
			return err != nil
		}()},
		{name: "MultiAggregate", col: true, rejected: func() bool {
			_, err := eng.(metaengine.MultiAggregateReader).MultiAggregate(ctx, "c", specs, filters)
			return err != nil
		}()},
		{name: "MultiGroupedAggregate", col: true, grp: true, rejected: func() bool {
			_, err := eng.(metaengine.MultiGroupedAggregateReader).MultiGroupedAggregate(
				ctx,
				"c",
				specs,
				groupBy,
				filters,
			)
			return err != nil
		}()},
		{name: "DistinctValues", col: true, rejected: func() bool {
			_, err := eng.(metaengine.DistinctReader).DistinctValues(ctx, "c", column, filters)
			return err != nil
		}()},
		{name: "ScanRawValues", col: true, grp: true, rejected: func() bool {
			_, err := eng.(metaengine.RawScanReader).ScanRawValues(ctx, "c", filters, nil, nil, 10)
			return err != nil
		}()},
		{name: "PushdownMapScan", col: true, grp: true, rejected: func() bool {
			_, err := eng.(interface {
				PushdownMapScan(context.Context, string, []metaengine.FilterSpec, *metaengine.SortSpec, any, int) (metaengine.ScanResult, error)
			}).
				PushdownMapScan(ctx, "c", filters, nil, nil, 10)
			return err != nil
		}()},
		{name: "ExplainScanQuery", col: true, grp: true, rejected: func() bool {
			sql, _ := eng.(metaengine.ExplainableScan).
				ExplainScanQuery(ctx, "c", metaengine.ExplainOptions{Filters: filters})
			return strings.HasPrefix(sql, "/* rejected:")
		}()},
		{name: "ExplainAggregateQuery", col: true, grp: true, rejected: func() bool {
			opts := metaengine.ExplainAggregateOptions{
				Fn: metaengine.AggregateSum, Column: column, GroupBy: groupBy, Filters: filters,
			}
			sql, _ := eng.(metaengine.ExplainableAggregate).ExplainAggregateQuery(ctx, "c", opts)

			return strings.HasPrefix(sql, "/* rejected:")
		}()},
	}

	return probes
}

func rejectedSQL(sql string) bool {
	return strings.HasPrefix(sql, "/* rejected:")
}

func assertAllRejected(t *testing.T, label string, scope surfaceScope, probes []surfaceProbe) {
	t.Helper()

	for _, p := range probes {
		if scope.matches(p) && !p.rejected {
			t.Errorf("%s: %s accepted hostile query input", label, p.name)
		}
	}
}

// newProbeEngine returns an engine with collection "c" populated, optionally
// planned so the planned-table builders are exercised as well.
func newProbeEngine(t *testing.T, planned bool) metaengine.Engine {
	t.Helper()

	eng := newTxTestEngine(t)

	if planned {
		if err := eng.(interface {
			ApplyLayout(string, []string, []string) error
		}).ApplyLayout("c", []string{"status"}, []string{"n"}); err != nil {
			t.Fatalf("ApplyLayout: %v", err)
		}
	}

	if err := eng.(metaengine.MapBackend).MapSet(context.Background(), "c", "k",
		map[string]any{"status": "open", "n": 1}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	return eng
}

func TestQuerySurfacesRejectHostileFilters(t *testing.T) {
	t.Parallel()

	for _, planned := range []bool{false, true} {
		eng := newProbeEngine(t, planned)

		for _, in := range hostileQueryInputs {
			label := fmt.Sprintf("planned=%v op=%q column=%q", planned, in.Op, in.Column)
			assertAllRejected(t, label, allSurfaces, probeQuerySurfaces(
				eng, context.Background(), "n", "status", []metaengine.FilterSpec{in}))
		}
	}
}

func TestQuerySurfacesRejectHostileColumns(t *testing.T) {
	t.Parallel()

	for _, planned := range []bool{false, true} {
		eng := newProbeEngine(t, planned)
		ctx := context.Background()

		for _, c := range hostileColumns {
			assertAllRejected(t, fmt.Sprintf("planned=%v column %q", planned, c),
				columnOnly, probeQuerySurfaces(eng, ctx, c, "status", nil))
			assertAllRejected(t, fmt.Sprintf("planned=%v groupBy %q", planned, c),
				groupByOnly, probeQuerySurfaces(eng, ctx, "n", c, nil))
		}

		_, err := eng.(metaengine.RawScanReader).ScanRawValues(ctx, "c", nil,
			&metaengine.SortSpec{Column: "n; DROP"}, nil, 10)
		if err == nil {
			t.Errorf("planned=%v ScanRawValues accepted a hostile sort column", planned)
		}
	}
}
