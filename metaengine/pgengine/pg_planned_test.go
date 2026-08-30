//go:build integration

package pgengine_test

// Live-Postgres tests for the pgengine planned-table path (D1): plans are
// registered via the LayoutPlanApplier capability, Map ops route to the
// planned table, and extracted-column type mismatches fail loudly.

import (
	"context"
	"errors"
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
