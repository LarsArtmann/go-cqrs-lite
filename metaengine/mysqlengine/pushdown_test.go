package mysqlengine_test

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestMySQLPushdownMapScan_FilterSortLimit(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	ps := eng.(metaengine.PushdownScan)

	items := []struct {
		key   string
		value map[string]any
	}{
		{"t1", map[string]any{"status": "open", "priority": float64(3)}},
		{"t2", map[string]any{"status": "open", "priority": float64(1)}},
		{"t3", map[string]any{"status": "done", "priority": float64(2)}},
		{"t4", map[string]any{"status": "open", "priority": float64(5)}},
		{"t5", map[string]any{"status": "done", "priority": float64(4)}},
	}

	for _, item := range items {
		g := gomega.NewWithT(t)
		g.Expect(mb.MapSet(ctx, "pushdown_test", item.key, item.value)).To(gomega.Succeed())
	}

	// Filter: status = open → 3 items.
	results, err := ps.PushdownMapScan(ctx, "pushdown_test",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		nil, nil, 0,
	)
	g := gomega.NewWithT(t)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.HaveLen(3))

	// Sort DESC by priority + limit 2.
	results, err = ps.PushdownMapScan(ctx, "pushdown_test",
		nil,
		&metaengine.SortSpec{Column: "priority", Desc: true},
		nil, 2,
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.HaveLen(2))
}

func TestMySQLPushdownMapScan_Combined(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	ps := eng.(metaengine.PushdownScan)

	for i, p := range []float64{3, 1, 5, 2, 4} {
		key := string(rune('a' + i))
		g := gomega.NewWithT(t)
		g.Expect(mb.MapSet(ctx, "combined_test", key, map[string]any{
			"status":   "open",
			"priority": p,
		})).To(gomega.Succeed())
	}

	results, err := ps.PushdownMapScan(ctx, "combined_test",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		&metaengine.SortSpec{Column: "priority", Desc: true},
		nil, 3,
	)
	g := gomega.NewWithT(t)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.HaveLen(3))

	priorities := extractFloatField(results.Items, "priority")
	g.Expect(priorities).To(gomega.Equal([]float64{5, 4, 3}))
}

func TestMySQLPushdownMapScan_EmptyResult(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(context.Background(), "empty_pushdown",
		[]metaengine.FilterSpec{{Column: "x", Op: metaengine.FilterEq, Value: "none"}},
		nil, nil, 0,
	)
	g := gomega.NewWithT(t)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.BeEmpty())
}

func extractFloatField(items []any, field string) []float64 {
	var vals []float64
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if v, ok := m[field].(float64); ok {
				vals = append(vals, v)
			}
		}
	}
	return vals
}
