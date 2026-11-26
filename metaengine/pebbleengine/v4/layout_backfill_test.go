package pebbleengine_test

import (
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPebbleApplyLayout_BackfillsExistingRows writes rows BEFORE the layout
// and checks that both the filter index and the sort index find them.
func TestPebbleApplyLayout_BackfillsExistingRows(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	ctx := fixtureCtx()
	mb := eng.(metaengine.MapBackend)
	keys := []string{"a", "b", "c"}
	rsr := eng.(metaengine.RawScanReader)

	for i, status := range []string{"active", "active", "archived"} {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "tasks", keys[i], map[string]any{
			"status": status, "score": 3,
		})).To(gomega.Succeed())
	}

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("tasks", []string{"status"}, []string{"score"})).To(
		gomega.Succeed())

	active := []metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "active"}}
	got, err := rsr.ScanRawValues(ctx, "tasks", active, nil, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(got.Items).To(gomega.HaveLen(2))

	sorted, err := rsr.ScanRawValues(ctx, "tasks", nil,
		&metaengine.SortSpec{Column: "score"}, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(sorted.Items).To(gomega.HaveLen(3))
}
