package mysqlengine_test

// Live-MariaDB/MySQL tests for the planned-table path (D2): plans registered
// via the LayoutPlanApplier capability route Map ops to extracted-column
// tables. Skips unless MYSQL_TEST_DSN is set.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// cleanupPlannedCollection removes a planned table and its meta_map rows so
// re-runs against the persistent cqrs_test database start clean.
func cleanupPlannedCollection(t *testing.T, dsn, collection string) {
	t.Helper()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("cleanup open: %v", err)
	}

	defer db.Close()

	table := "meta_planned_" + sanitizeCollectionName(collection)
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS `" + table + "`",
		"DELETE FROM meta_map WHERE collection = ?",
	} {
		if stmt == "DELETE FROM meta_map WHERE collection = ?" {
			if _, err := db.Exec(stmt, collection); err != nil {
				t.Fatalf("cleanup meta_map: %v", err)
			}

			return
		}

		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("cleanup drop: %v", err)
		}
	}
}

func sanitizeCollectionName(c string) string {
	out := make([]rune, 0, len(c))
	for _, r := range c {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}

	return string(out)
}

// TestMySQLPlannedTable_RoundTrip pins the Map routing through the planned
// table: upsert, get, delete, and conflict rejection on live MariaDB.
func TestMySQLPlannedTable_RoundTrip(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "planned_roundtrip")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "planned_roundtrip") })
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
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
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

// TestMySQLPlannedPushdownScan_FilterSortKeyset pins D3 slice 1 on
// MariaDB: after ApplyLayoutPlan, PushdownMapScan reads the extracted-column
// table — native-column filters, ASC/DESC sort, keyset cursor pagination,
// has-more, build-time mis-type Rejection — while planless collections keep
// the meta_map path.
func TestMySQLPlannedPushdownScan_FilterSortKeyset(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "planned_scan")
	cleanupPlannedCollection(t, mysqlTestDSN(), "plain_scan")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "plain_scan") })
	mb := eng.(metaengine.MapBackend)
	ps, ok := eng.(metaengine.PushdownScan)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement PushdownScan")
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_scan", []string{"status"}, []string{"priority"}),
	)).To(gomega.Succeed())

	for i := 1; i <= 5; i++ {
		status := "open"
		if i%2 == 0 {
			status = "done"
		}

		g.Expect(mb.MapSet(
			ctx,
			"planned_scan",
			fmt.Sprintf("k%d", i),
			map[string]any{
				"priority": float64(i),
				"status":   status,
				"title":    "t",
			},
		)).To(gomega.Succeed())
	}

	res, err := ps.PushdownMapScan(ctx, "planned_scan",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		&metaengine.SortSpec{Column: "priority"}, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(3))
	g.Expect(res.Items[0].(map[string]any)["priority"]).To(gomega.Equal(float64(1)))

	page1, err := ps.PushdownMapScan(ctx, "planned_scan", nil,
		&metaengine.SortSpec{Column: "priority", Desc: true}, nil, 2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(page1.Items).To(gomega.HaveLen(2))
	g.Expect(page1.HasMore).To(gomega.BeTrue())

	lastPrio := page1.Items[1].(map[string]any)["priority"]

	page2, err := ps.PushdownMapScan(ctx, "planned_scan", nil,
		&metaengine.SortSpec{Column: "priority", Desc: true}, lastPrio, 2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(page2.Items).NotTo(gomega.BeEmpty())
	g.Expect(page2.Items[0].(map[string]any)["priority"]).To(gomega.Equal(float64(3)))

	_, err = ps.PushdownMapScan(ctx, "planned_scan",
		[]metaengine.FilterSpec{{Column: "priority", Op: metaengine.FilterEq, Value: "high"}},
		nil, nil, 0)
	g.Expect(errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch)).To(gomega.BeTrue())

	g.Expect(mb.MapSet(ctx, "plain_scan", "a", map[string]any{"v": float64(1)})).
		To(gomega.Succeed())
	plain, err := ps.PushdownMapScan(ctx, "plain_scan", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(plain.Items).To(gomega.HaveLen(1))
}

// TestMySQLPlannedMapScan_VisibilityParity pins D3 slice 2 (read side).
func TestMySQLPlannedMapScan_VisibilityParity(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "planned_vis")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "planned_vis") })
	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.ScanBackend)
	lpa := eng.(metaengine.LayoutPlanApplier)

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_vis", []string{"kind"}, nil),
	)).To(gomega.Succeed())

	g.Expect(mb.MapSet(ctx, "planned_vis", "k1", map[string]any{"kind": "a"})).To(gomega.Succeed())
	g.Expect(mb.MapSet(ctx, "planned_vis", "k2", map[string]any{"kind": "b"})).To(gomega.Succeed())

	res, err := sb.MapScan(ctx, "planned_vis", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(2))

	g.Expect(mb.MapDelete(ctx, "planned_vis", "k1")).To(gomega.Succeed())

	res, err = sb.MapScan(ctx, "planned_vis", nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(1))
}

// TestMySQLPlannedMapUpdate_RoundTrip pins D3 slice 2 (write side): the
// read-modify-write hits the planned table, recomputes extracted columns,
// creates on a missing key (nil prev), and works inside RunInTx.
func TestMySQLPlannedMapUpdate_RoundTrip(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "planned_upd")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "planned_upd") })
	mb := eng.(metaengine.MapBackend)
	mu, ok := eng.(metaengine.MapUpdater)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement MapUpdater")
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

	g.Expect(mu.MapUpdate(ctx, "planned_upd", "new", func(prev any) any {
		g.Expect(prev).To(gomega.BeNil())

		return map[string]any{"count": float64(10)}
	})).To(gomega.Succeed())

	got, found, err = mb.MapGet(ctx, "planned_upd", "new")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got).To(gomega.Equal(map[string]any{"count": float64(10)}))

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

// TestMySQLPlannedFromType_FloatColumnsAreNumeric pins the sqlTypeOf fix on
// MariaDB: a float64 field produces a DOUBLE extracted column, so numeric
// filters compare numerically.
func TestMySQLPlannedFromType_FloatColumnsAreNumeric(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	cleanupPlannedCollection(t, mysqlTestDSN(), "planned_float")
	t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), "planned_float") })
	mb := eng.(metaengine.MapBackend)
	ps := eng.(metaengine.PushdownScan)
	lpa := eng.(metaengine.LayoutPlanApplier)

	type taskRow struct {
		Score float64 `json:"score"`
		State string  `json:"state"`
	}

	plan := metaengine.BuildLayoutPlanFromType[taskRow](
		"planned_float",
		[]string{"state"},
		[]string{"score"},
	)
	g.Expect(lpa.ApplyLayoutPlan(plan)).To(gomega.Succeed())

	for i, score := range []float64{0.25, 9.5, 3.75} {
		g.Expect(mb.MapSet(ctx, "planned_float", fmt.Sprintf("k%d", i),
			map[string]any{"score": score, "state": "open"})).To(gomega.Succeed())
	}

	res, err := ps.PushdownMapScan(ctx, "planned_float",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGt, Value: 3.0}},
		&metaengine.SortSpec{Column: "score"}, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res.Items).To(gomega.HaveLen(2))
	g.Expect(res.Items[0].(map[string]any)["score"]).To(gomega.Equal(3.75))
	g.Expect(res.Items[1].(map[string]any)["score"]).To(gomega.Equal(9.5))
}
