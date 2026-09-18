package mysqlengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
)

// TestMySQLBackfillPlannedCollection pins the opt-in backfill helper on live
// MariaDB/MySQL: rows written to meta_map before the plan was registered
// become visible to planned scans, keyset paging covers multi-batch
// collections, extracted columns are recomputed (numeric predicates work),
// and re-running is idempotent.
func TestMySQLBackfillPlannedCollection(t *testing.T) {
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "backfill_items")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "backfill_items") })

	mb := eng.(metaengine.MapBackend)

	// Seed meta_map while the collection is still planless.
	for i := range 7 {
		doc := map[string]any{"status": "open", "amount": float64(i)}
		if i%2 == 1 {
			doc["status"] = "done"
		}

		g.Expect(mb.MapSet(ctx, "backfill_items", fmt.Sprintf("k%02d", i), doc)).
			To(gomega.Succeed())
	}

	lpa := eng.(metaengine.LayoutPlanApplier)
	plan := metaengine.BuildLayoutPlan("backfill_items", []string{"status"}, []string{"amount"})
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	// Pre-registration rows are invisible until backfilled.
	scan := eng.(metaengine.ScanBackend)
	res, err := scan.MapScan(ctx, "backfill_items", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.BeEmpty(), "no-backfill contract holds before explicit backfill")

	// Backfill with paging (batch 3 over 7 rows).
	n, err := metaengine.BackfillPlannedCollection(ctx, eng, "backfill_items", 3)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(n).To(gomega.Equal(7))

	// Idempotency: re-running re-upserts the same rows.
	n2, err := metaengine.BackfillPlannedCollection(ctx, eng, "backfill_items", 3)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(n2).To(gomega.Equal(7))

	// Planned scans now see everything, with recomputed extracted columns.
	res, err = scan.MapScan(ctx, "backfill_items", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(7))

	pushdown := eng.(metaengine.PushdownScan)
	res, err = pushdown.PushdownMapScan(
		ctx, "backfill_items",
		[]metaengine.FilterSpec{{Column: "amount", Op: metaengine.FilterGt, Value: 5.5}},
		nil, nil, 0,
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(1))
}
