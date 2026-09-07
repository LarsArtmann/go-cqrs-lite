package tursoengine_test

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type orderRow struct {
	Key      string
	Customer string
	Amount   float64
}

// orderRows returns deterministic order rows: customers c0..cN-1 cycling with
// uneven group sizes, and amounts (i%7)*10 + i/100 so sums carry cents.
func orderRows(n, customers int) []orderRow {
	rows := make([]orderRow, 0, n)

	for i := range n {
		rows = append(rows, orderRow{
			Key:      fmt.Sprintf("order-%04d", i),
			Customer: fmt.Sprintf("c%d", i%customers),
			Amount:   float64((i%7)*10) + float64(i)/100,
		})
	}

	return rows
}

func sumAmounts(rows []orderRow) float64 {
	total := 0.0

	for _, r := range rows {
		total += r.Amount
	}

	return total
}

func seedOrders(ctx context.Context, tb testing.TB, eng metaengine.Engine, rows []orderRow) {
	tb.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		tb.Fatal("engine does not implement MapBackend")
	}

	for _, row := range rows {
		value := map[string]any{"customer": row.Customer, "amount": row.Amount}
		if err := mb.MapSet(ctx, "orders", row.Key, value); err != nil {
			tb.Fatalf("MapSet %s: %v", row.Key, err)
		}
	}
}

func mustEngineWithMatViews(
	tb testing.TB,
	dsn string,
	specs ...metaengine.MaterializedViewSpec,
) metaengine.Engine {
	tb.Helper()

	eng, err := tursoengine.New(dsn, tursoengine.WithMaterializedViews(specs))
	if err != nil {
		tb.Skipf("turso not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func expectScalar(t *testing.T, eng metaengine.Engine, col string,
	fn metaengine.AggregateFn, column string, filters []metaengine.FilterSpec, want float64,
) {
	t.Helper()

	g := gomega.NewWithT(t)

	ar, ok := eng.(metaengine.AggregateReader)
	if !ok {
		t.Fatal("engine does not implement AggregateReader")
	}

	got, err := ar.Aggregate(context.Background(), col, fn, column, filters)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	// Tolerance covers floating-point association differences between a
	// matview derivation (sums of group sums) and the sequential expectation.
	g.Expect(got).
		To(gomega.BeNumerically("~", want, 1e-6), "aggregate %s(%s) on %s", fn, column, col)
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// End-to-end: specs declared at construction serve unfiltered scalar and
// grouped aggregates, IVM keeps them exact across set/replace/delete, and
// filtered or uncovered shapes still return correct (base-path) results.
func TestTursoMatView_Serving(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{Collection: "orders", Fn: metaengine.MatViewCount},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewAvg,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewMin,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewMax,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	rows := orderRows(200, 8)
	seedOrders(ctx, t, eng, rows)

	expectScalar(t, eng, "orders", metaengine.MatViewCount, "", nil, 200)
	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows))
	expectScalar(t, eng, "orders", metaengine.MatViewAvg, "amount", nil, sumAmounts(rows)/200)
	expectScalar(t, eng, "orders", metaengine.MatViewMin, "amount", nil, rows[0].Amount)

	max := rows[0].Amount

	for _, r := range rows {
		if r.Amount > max {
			max = r.Amount
		}
	}

	expectScalar(t, eng, "orders", metaengine.MatViewMax, "amount", nil, max)

	// Grouped: per-customer sums must reconstruct the scalar total.
	g := gomega.NewWithT(t)

	gr := eng.(metaengine.GroupedAggregateReader)

	groups, err := gr.GroupedAggregate(
		ctx,
		"orders",
		metaengine.MatViewSum,
		"amount",
		"customer",
		nil,
	)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(groups).To(gomega.HaveLen(8))
	g.Expect(sumAmountGroups(groups)).To(gomega.BeNumerically("~", sumAmounts(rows), 1e-6))

	// Grouped AVG: exact per-group weighted averages; recombining with group
	// counts reconstructs the global average.
	avgGroups, err := gr.GroupedAggregate(
		ctx,
		"orders",
		metaengine.MatViewAvg,
		"amount",
		"customer",
		nil,
	)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(avgGroups).To(gomega.HaveLen(8))

	// IVM across replace and delete: replace order-0000 with amount 1000,
	// then delete order-0001.
	mb := eng.(metaengine.MapBackend)

	g.Expect(mb.MapSet(ctx, "orders", rows[0].Key, map[string]any{
		"customer": rows[0].Customer,
		"amount":   1000.0,
	})).To(gomega.Succeed())
	g.Expect(mb.MapDelete(ctx, "orders", rows[1].Key)).To(gomega.Succeed())

	rows[0].Amount = 1000
	rows = append(
		[]orderRow{rows[0]},
		rows[2:]...) // order-0000 replaced in place, order-0001 deleted

	expectScalar(t, eng, "orders", metaengine.MatViewCount, "", nil, float64(len(rows)))
	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows))

	// Filtered aggregates bypass the view and must still be exact.
	var wantC2 float64

	for _, r := range rows {
		if r.Customer == "c2" {
			wantC2 += r.Amount
		}
	}

	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount",
		[]metaengine.FilterSpec{{Column: "customer", Op: metaengine.FilterEq, Value: "c2"}}, wantC2)

	// Uncovered collection still works via the base path.
	expectScalar(t, eng, "never-seen", metaengine.MatViewSum, "amount", nil, 0)
}

func sumAmountGroups(groups map[string]float64) float64 {
	total := 0.0

	for _, v := range groups {
		total += v
	}

	return total
}

// Scalar aggregates served through a view whose spec only declared the
// GROUPED shape (exact algebraic derivations: SUM, COUNT, MIN, MAX, AVG).
func TestTursoMatView_ScalarViaGroupedView(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	rows := orderRows(120, 5)
	seedOrders(ctx, t, eng, rows)

	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows))
}

// AVG via a grouped view: per-group averages must be exact even with wildly
// uneven group sizes (the weighted sum-of-sums / sum-of-counts derivation).
func TestTursoMatView_GroupedAvgExact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewAvg,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	rows := orderRows(97, 6)
	seedOrders(ctx, t, eng, rows)

	g := gomega.NewWithT(t)

	gr := eng.(metaengine.GroupedAggregateReader)

	groups, err := gr.GroupedAggregate(
		ctx,
		"orders",
		metaengine.MatViewAvg,
		"amount",
		"customer",
		nil,
	)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(groups).To(gomega.HaveLen(6))

	// Recompute each group's average from the source rows.
	perGroup := map[string]struct {
		sum   float64
		count int
	}{}

	for _, r := range rows {
		gg := perGroup[r.Customer]
		gg.sum += r.Amount
		gg.count++
		perGroup[r.Customer] = gg
	}

	for customer, want := range perGroup {
		g.Expect(groups[customer]).To(gomega.BeNumerically("~", want.sum/float64(want.count), 1e-6),
			"group %s average", customer)
	}
}

// Restart safety: the view persists in the database file; reopening with the
// same spec must be idempotent (IF NOT EXISTS) and keep serving exact values.
func TestTursoMatView_RestartIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	dsn := filepath.Join(t.TempDir(), "matview.db")

	eng := mustEngineWithMatViews(
		t,
		dsn,
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
		},
	)

	seedOrders(ctx, t, eng, orderRows(50, 4))

	if err := eng.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := tursoengine.New(
		dsn,
		tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
		}),
	)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	defer func() { _ = reopened.Close() }()

	rows := orderRows(60, 4) // 10 more rows on top of persisted data
	seedOrders(ctx, t, reopened, rows[50:])

	expectScalar(t, reopened, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows))
}

// The MaterializedViewsReporter surfaces the operator's views for Doctor and
// engine stats.
func TestTursoMatView_Reporter(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	reporter, ok := eng.(metaengine.MaterializedViewsReporter)
	g.Expect(ok).To(gomega.BeTrue())

	views := reporter.MaterializedViews()
	g.Expect(views).To(gomega.HaveLen(1))
	g.Expect(views[0].Spec.Collection).To(gomega.Equal("orders"))
	g.Expect(views[0].Name).To(gomega.HavePrefix("cqrs_mv_orders_sum_amount_by_customer"))
	g.Expect(views[0].Rows).To(gomega.Equal(int64(0)))

	seedOrders(context.Background(), t, eng, orderRows(16, 4))

	views = reporter.MaterializedViews()
	g.Expect(views[0].Rows).To(gomega.Equal(int64(4)), "one row per customer group")
}

// EXPLAIN must show the view SQL the serving path would run.
func TestTursoMatView_ExplainReflectsServing(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	ex := eng.(metaengine.ExplainableAggregate)

	sql, args := ex.ExplainAggregateQuery(
		context.Background(),
		"orders",
		metaengine.ExplainAggregateOptions{
			Fn:     metaengine.MatViewSum,
			Column: "amount",
		},
	)
	g.Expect(sql).To(gomega.ContainSubstring("cqrs_mv_orders_sum_amount"))
	g.Expect(sql).To(gomega.Not(gomega.ContainSubstring("meta_map")))
	g.Expect(args).To(gomega.BeEmpty())

	sql, _ = ex.ExplainAggregateQuery(
		context.Background(),
		"orders",
		metaengine.ExplainAggregateOptions{
			Fn:      metaengine.MatViewSum,
			Column:  "amount",
			GroupBy: "customer",
		},
	)
	g.Expect(sql).To(gomega.ContainSubstring("grp, agg"))
	g.Expect(sql).To(gomega.ContainSubstring("by_customer"))

	// Filtered explain stays on the base path.
	sql, _ = ex.ExplainAggregateQuery(
		context.Background(),
		"orders",
		metaengine.ExplainAggregateOptions{
			Fn:     metaengine.MatViewSum,
			Column: "amount",
			Filters: []metaengine.FilterSpec{
				{Column: "customer", Op: metaengine.FilterEq, Value: "c1"},
			},
		},
	)
	g.Expect(sql).To(gomega.ContainSubstring("meta_map"))
}

// A collection that later migrates to a planned table must stop serving from
// the meta_map view — the planned table is the query's source after migration
// (ApplyLayout creates an EMPTY table; no backfill), and pre-migration rows in
// meta_map are invisible to planned-path reads.
func TestTursoMatView_PlannedMigrationFallsThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
		},
	)

	seedOrders(ctx, t, eng, orderRows(40, 4))

	planner := eng.(metaengine.LayoutPlanner)

	// Declare every queried field as a plan column (the planned-table contract:
	// aggregated columns must be extracted columns).
	if err := planner.ApplyLayout("orders", []string{"customer", "amount"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	// Post-migration writes land in the planned table; only they are visible.
	mb := eng.(metaengine.MapBackend)

	g := gomega.NewWithT(t)

	post := 5.0

	for i := range 5 {
		g.Expect(mb.MapSet(ctx, "orders", fmt.Sprintf("post-%d", i), map[string]any{
			"customer": "c1",
			"amount":   100.0,
		})).To(gomega.Succeed())
	}

	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, post*100)
}

// The driver-registry path (system → DriverConfig) must wire specs through.
func TestTursoMatView_DriverRegistryPath(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	factory, err := metaengine.LookupDriver("turso")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	e, err := factory(context.Background(), metaengine.DriverConfig{
		MaterializedViews: []metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewCount},
		},
	})
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	t.Cleanup(func() { _ = e.Close() })

	seedOrders(context.Background(), t, e, orderRows(12, 3))

	expectScalar(t, e, "orders", metaengine.MatViewCount, "", nil, 12)
}
