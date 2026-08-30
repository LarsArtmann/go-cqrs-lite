//go:build integration

package pgengine_test

// Live-Postgres tests for the pgengine planned-table path (D1): plans are
// registered via the LayoutPlanApplier capability, Map ops route to the
// planned table, and extracted-column type mismatches fail loudly.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPgPlannedTable_RoundTrip pins the Map routing: after ApplyLayoutPlan,
// MapSet/MapGet/MapDelete hit the planned table with extracted columns and
// the value round-trips. Conflicting re-registrations and unplanned
// collections are unaffected.
func TestPgPlannedTable_RoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa, ok := eng.(metaengine.LayoutPlanApplier)
	g.Expect(ok).To(gomega.BeTrue(), "pgEngine must implement LayoutPlanApplier")

	plan := metaengine.BuildLayoutPlan("planned_roundtrip", []string{"priority"}, nil)
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	// Idempotent: same columns → no-op.
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	doc := map[string]any{"priority": float64(7), "title": "hello"}
	g.Expect(mb.MapSet(ctx, "planned_roundtrip", "k1", doc)).To(gomega.Succeed())

	got, found, err := mb.MapGet(ctx, "planned_roundtrip", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got).To(gomega.Equal(map[string]any{"priority": float64(7), "title": "hello"}))

	// Unplanned collections are untouched.
	_, foundUnplanned, err := mb.MapGet(ctx, "planned_other", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundUnplanned).To(gomega.BeFalse())

	g.Expect(mb.MapDelete(ctx, "planned_roundtrip", "k1")).To(gomega.Succeed())

	_, foundAfter, err := mb.MapGet(ctx, "planned_roundtrip", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundAfter).To(gomega.BeFalse())
}

// TestPgPlannedTable_ConflictingPlanRejected pins the conflict guard:
// re-registering the same collection with DIFFERENT columns fails with
// ErrLayoutConflict instead of silently re-creating the table.
func TestPgPlannedTable_ConflictingPlanRejected(t *testing.T) {
	g := gomega.NewWithT(t)

	eng := mustNewPgEngine(t)
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_conflict", []string{"status"}, nil),
	)).To(gomega.Succeed())

	err := lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_conflict", []string{"region"}, nil),
	)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(errors.Is(err, metaengine.ErrLayoutConflict)).To(gomega.BeTrue())
}

// TestPgPlannedTable_MisTypedExtractFailsLoudly pins the fail-loud contract
// for extracted columns: a document whose field type contradicts the
// extracted BIGINT column is rejected by Postgres instead of silently
// succeeding (the JSONB value column would have accepted anything).
func TestPgPlannedTable_MisTypedExtractFailsLoudly(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_mistype", []string{"priority"}, nil),
	)).To(gomega.Succeed())

	// "priority" is a BIGINT extracted column; a string payload must fail.
	g.Expect(mb.MapSet(ctx, "planned_mistype", "bad",
		map[string]any{"priority": "not-a-number"})).NotTo(gomega.Succeed())
}

// TestPgPlannedPushdownScan_FilterSortKeyset pins D3 slice 1: after
// ApplyLayoutPlan, PushdownMapScan reads the extracted-column table —
// native-column filters, ASC/DESC sort, keyset cursor pagination, and
// has-more detection — while planless collections keep the meta_map path.
func TestPgPlannedPushdownScan_FilterSortKeyset(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	ps, ok := eng.(metaengine.PushdownScan)
	g.Expect(ok).To(gomega.BeTrue(), "pgEngine must implement PushdownMapScanner")
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_scan", []string{"status"}, []string{"priority"}),
	)).To(gomega.Succeed())

	for i := 1; i <= 5; i++ {
		status := "open"
		if i%2 == 0 {
			status = "done"
		}

		g.Expect(mb.MapSet(ctx, "planned_scan", fmt.Sprintf("k%d", i),
			map[string]any{"priority": float64(i), "status": status, "title": "t"})).To(gomega.Succeed())
	}

	// Filter + ASC sort through the planned table.
	res, err := ps.PushdownMapScan(ctx, "planned_scan",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		&metaengine.SortSpec{Column: "priority"}, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(3))

	first := res.Items[0].(map[string]any)
	g.Expect(first["priority"]).To(gomega.Equal(float64(1)))

	// DESC sort + keyset pagination with has-more.
	page1, err := ps.PushdownMapScan(ctx, "planned_scan", nil,
		&metaengine.SortSpec{Column: "priority", Desc: true}, nil, 2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(page1.Items).To(gomega.HaveLen(2))
	g.Expect(page1.HasMore).To(gomega.BeTrue())
	g.Expect(page1.Items[0].(map[string]any)["priority"]).To(gomega.Equal(float64(5)))

	lastPrio := page1.Items[1].(map[string]any)["priority"]

	page2, err := ps.PushdownMapScan(ctx, "planned_scan", nil,
		&metaengine.SortSpec{Column: "priority", Desc: true}, lastPrio, 2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(page2.Items).NotTo(gomega.BeEmpty())
	g.Expect(page2.Items[0].(map[string]any)["priority"]).To(gomega.Equal(float64(3)))

	// Mis-typed filter values are a classified Rejection (build-time).
	_, err = ps.PushdownMapScan(ctx, "planned_scan",
		[]metaengine.FilterSpec{{Column: "priority", Op: metaengine.FilterEq, Value: "high"}},
		nil, nil, 0)
	g.Expect(errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch)).To(gomega.BeTrue())

	// Planless collections still route meta_map (fallback intact).
	g.Expect(mb.MapSet(ctx, "plain_scan", "a", map[string]any{"v": float64(1)})).To(gomega.Succeed())
	plain, err := ps.PushdownMapScan(ctx, "plain_scan", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(plain.Items).To(gomega.HaveLen(1))
}

// TestPgPlannedMapScan_VisibilityParity pins D3 slice 2 (read side): rows
// written through the planned path are visible to MapScan (the closure
// fallback) — the documented meta_map/planned visibility split is closed.
func TestPgPlannedMapScan_VisibilityParity(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_vis", []string{"kind"}, nil),
	)).To(gomega.Succeed())

	g.Expect(mb.MapSet(ctx, "planned_vis", "k1", map[string]any{"kind": "a"})).To(gomega.Succeed())
	g.Expect(mb.MapSet(ctx, "planned_vis", "k2", map[string]any{"kind": "b"})).To(gomega.Succeed())

	sb := eng.(metaengine.ScanBackend)

	res, err := sb.MapScan(ctx, "planned_vis", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(2))

	g.Expect(mb.MapDelete(ctx, "planned_vis", "k1")).To(gomega.Succeed())

	res, err = sb.MapScan(ctx, "planned_vis", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(1))
}

// TestPgPlannedMapUpdate_RoundTrip pins D3 slice 2 (write side): the
// read-modify-write hits the planned table, recomputes extracted columns,
// creates on a missing key (nil prev), and works inside RunInTx.
func TestPgPlannedMapUpdate_RoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	mu, ok := eng.(metaengine.MapUpdater)
	g.Expect(ok).To(gomega.BeTrue(), "pgEngine must implement MapUpdater")
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_upd", []string{"count"}, nil),
	)).To(gomega.Succeed())

	g.Expect(mb.MapSet(ctx, "planned_upd", "k1",
		map[string]any{"count": float64(1), "name": "x"})).To(gomega.Succeed())

	g.Expect(mu.MapUpdate(ctx, "planned_upd", "k1", func(prev any) any {
		p := prev.(map[string]any)
		p["count"] = p["count"].(float64) + 1

		return p
	})).To(gomega.Succeed())

	got, found, err := mb.MapGet(ctx, "planned_upd", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got).To(gomega.Equal(map[string]any{"count": float64(2), "name": "x"}))

	// Missing key: prev is nil, update creates the row.
	g.Expect(mu.MapUpdate(ctx, "planned_upd", "new", func(prev any) any {
		g.Expect(prev).To(gomega.BeNil())

		return map[string]any{"count": float64(10)}
	})).To(gomega.Succeed())

	got, found, err = mb.MapGet(ctx, "planned_upd", "new")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got).To(gomega.Equal(map[string]any{"count": float64(10)}))

	// Inside RunInTx: participates in the outer transaction.
	g.Expect(eng.(metaengine.Transactional).RunInTx(ctx, func(ctx context.Context) error {
		return mu.MapUpdate(ctx, "planned_upd", "k1", func(prev any) any {
			p := prev.(map[string]any)
			p["count"] = p["count"].(float64) + 1

			return p
		})
	})).To(gomega.Succeed())

	got, _, err = mb.MapGet(ctx, "planned_upd", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(got).To(gomega.Equal(map[string]any{"count": float64(3), "name": "x"}))
}
