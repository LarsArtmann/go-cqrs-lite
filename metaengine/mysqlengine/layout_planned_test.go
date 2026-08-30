package mysqlengine_test

// Live-MariaDB/MySQL tests for the planned-table path (D2): plans registered
// via the LayoutPlanApplier capability route Map ops to extracted-column
// tables. Skips unless MYSQL_TEST_DSN is set.

import (
	"context"
	"errors"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQLPlannedTable_RoundTrip pins the Map routing through the planned
// table: upsert, get, delete, and conflict rejection on live MariaDB.
func TestMySQLPlannedTable_RoundTrip(t *testing.T) {
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa, ok := eng.(metaengine.LayoutPlanApplier)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement LayoutPlanApplier")

	plan := metaengine.BuildLayoutPlan("planned_roundtrip", []string{"priority"}, nil)
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	// Idempotent: same columns → no-op.
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	doc := map[string]any{"priority": float64(7), "title": "hello"}
	g.Expect(mb.MapSet(ctx, "planned_roundtrip", "k1", doc)).To(gomega.Succeed())

	// Upsert: same key, new value.
	g.Expect(mb.MapSet(ctx, "planned_roundtrip", "k1",
		map[string]any{"priority": float64(9), "title": "updated"})).To(gomega.Succeed())

	got, found, err := mb.MapGet(ctx, "planned_roundtrip", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got).To(gomega.Equal(map[string]any{"priority": float64(9), "title": "updated"}))

	g.Expect(mb.MapDelete(ctx, "planned_roundtrip", "k1")).To(gomega.Succeed())

	_, foundAfter, err := mb.MapGet(ctx, "planned_roundtrip", "k1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundAfter).To(gomega.BeFalse())
}

// TestMySQLPlannedTable_ConflictingPlanRejected pins the conflict guard.
func TestMySQLPlannedTable_ConflictingPlanRejected(t *testing.T) {
	mariadbVersion(t)

	g := gomega.NewWithT(t)

	eng := mustNewMySQLEngine(t)
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
