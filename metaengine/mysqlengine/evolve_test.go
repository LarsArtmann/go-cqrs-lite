package mysqlengine_test

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQLEvolveLayoutPlan pins the information_schema evolution contract on
// live MariaDB/MySQL: missing columns are added, drifted types are retyped,
// re-running on a matching schema applies nothing (idempotency), and the
// evolved schema still serves planned numeric predicates.
func TestMySQLEvolveLayoutPlan(t *testing.T) {
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "evolve_items")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "evolve_items") })

	evolver, ok := eng.(metaengine.LayoutPlanEvolver)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement LayoutPlanEvolver")

	// v1: amount declared TEXT (the legacy mis-typed shape).
	v1 := metaengine.LayoutPlan{
		Collection: "evolve_items",
		Table:      "meta_planned_evolve_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "amount", Type: "TEXT"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_meta_planned_evolve_items_status", Columns: []string{"status"}},
		},
	}

	applied, err := evolver.EvolveLayoutPlan(ctx, v1)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(applied).To(gomega.BeEmpty(), "evolve on a fresh table applies nothing")

	// v2: amount retyped to DOUBLE — the evolution this capability exists for.
	v2 := metaengine.LayoutPlan{
		Collection: "evolve_items",
		Table:      "meta_planned_evolve_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "amount", Type: "DOUBLE"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_meta_planned_evolve_items_status", Columns: []string{"status"}},
			{Name: "idx_meta_planned_evolve_items_amount", Columns: []string{"amount"}},
		},
	}

	applied, err = evolver.EvolveLayoutPlan(ctx, v2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(applied).To(gomega.ContainElement("retype:amount"))

	// Idempotency: evolving again applies nothing.
	applied, err = evolver.EvolveLayoutPlan(ctx, v2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(applied).To(gomega.BeEmpty())

	// Adding a column to the registered plan.
	v3 := v2
	v3.Columns = append(v3.Columns, metaengine.PlannedColumn{Name: "qty", Type: "INTEGER"})
	v3.Indexes = append(v3.Indexes, metaengine.PlannedIndex{
		Name: "idx_meta_planned_evolve_items_qty", Columns: []string{"qty"},
	})

	applied, err = evolver.EvolveLayoutPlan(ctx, v3)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(applied).To(gomega.ContainElement("add:qty"))

	// The evolved planned table serves typed writes and numeric predicates.
	mb := eng.(metaengine.MapBackend)
	g.Expect(mb.MapSet(ctx, "evolve_items", "a",
		map[string]any{"status": "open", "amount": 2.5})).To(gomega.Succeed())

	pushdown := eng.(metaengine.PushdownScan)
	res, err := pushdown.PushdownMapScan(
		ctx, "evolve_items",
		[]metaengine.FilterSpec{{Column: "amount", Op: metaengine.FilterGt, Value: 2.0}},
		nil, nil, 0,
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(1))
}
