package mysqlengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
)

// TestMySQLPlannedTablesReporting pins the PlannedTablesReporter capability
// on live MariaDB/MySQL: a registered planned collection is listed with its
// physical table, extracted columns, and a live row count.
func TestMySQLPlannedTablesReporting(t *testing.T) {
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "report_items")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "report_items") })

	lpa := eng.(metaengine.LayoutPlanApplier)
	plan := metaengine.BuildLayoutPlan("report_items", []string{"status"}, nil)
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)
	g.Expect(mb.MapSet(ctx, "report_items", "k1", map[string]any{"status": "open"})).
		To(gomega.Succeed())

	reporter, ok := eng.(metaengine.PlannedTablesReporter)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement PlannedTablesReporter")

	infos, err := reporter.PlannedTables(ctx)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(infos).To(gomega.HaveLen(1))

	info := infos[0]
	g.Expect(info.Collection).To(gomega.Equal("report_items"))
	g.Expect(info.Table).To(gomega.Equal("meta_planned_report_items"))
	g.Expect(info.Rows).To(gomega.Equal(int64(1)))
	g.Expect(info.Columns).To(gomega.Equal([]string{"status"}))
}
