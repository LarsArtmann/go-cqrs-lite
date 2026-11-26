package pebbleengine_test

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPebbleFilterIndex_AppliesEveryFilter pins that the filter-index scan
// re-applies the filters the index did not narrow: only "status" is indexed,
// so "score" must still be evaluated in Go.
func TestPebbleFilterIndex_AppliesEveryFilter(t *testing.T) {
	t.Parallel()

	ctx := fixtureCtx()
	f := newLayoutFixture(t, "items", []string{"status"}, nil)

	for _, row := range []struct {
		key, status string
		score       int
	}{
		{"a", "active", 1}, {"b", "active", 9}, {"c", "active", 5}, {"d", "archived", 9},
	} {
		gom := gomega.NewWithT(t)
		gom.Expect(f.mb.MapSet(ctx, "items", row.key, map[string]any{
			"status": row.status, "score": row.score,
		})).To(gomega.Succeed())
	}

	rsr := f.eng.(metaengine.RawScanReader)
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
		{Column: "score", Op: metaengine.FilterGt, Value: 5},
	}

	// Indexed filter first: the status index narrows the range, score is Go-side.
	indexedFirst, err := rsr.ScanRawValues(ctx, "items", filters, nil, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(extractField[string](t, indexedFirst.Items, "status")).To(
		gomega.HaveLen(1))

	// The same filters in the other order take the same index path.
	reversed := []metaengine.FilterSpec{filters[1], filters[0]}
	got, err := rsr.ScanRawValues(ctx, "items", reversed, nil, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(extractField[string](t, got.Items, "status")).To(gomega.HaveLen(1))
	gomega.NewWithT(t).Expect(extractField[float64](t, got.Items, "score")).To(
		gomega.Equal([]float64{9}))
}

// TestPebbleFilterIndex_CursorRoundTrip pages with the cursor the engine
// itself hands back (a compound SortKeyCursor), on a sort column that is not
// the indexed one.
func TestPebbleFilterIndex_CursorRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := fixtureCtx()
	f := newLayoutFixture(t, "items", []string{"status"}, nil)

	for i := 1; i <= 5; i++ {
		gomega.NewWithT(t).Expect(f.mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"status": "active", "score": i,
		})).To(gomega.Succeed())
	}

	rsr := f.eng.(metaengine.RawScanReader)
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	for _, desc := range []bool{false, true} {
		want := []float64{1, 2, 3, 4, 5}
		if desc {
			want = []float64{5, 4, 3, 2, 1}
		}

		sortSpec := &metaengine.SortSpec{Column: "score", Desc: desc}

		var got []float64

		var cursor any

		for range 6 {
			page, err := rsr.ScanRawValues(ctx, "items", filters, sortSpec, cursor, 2)
			gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

			if len(page.Items) == 0 {
				break
			}

			got = append(got, extractField[float64](t, page.Items, "score")...)

			if !page.HasMore {
				break
			}

			gomega.NewWithT(t).Expect(page.NextCursor).NotTo(gomega.BeNil())

			cursor = page.NextCursor
		}

		gomega.NewWithT(t).Expect(got).To(gomega.Equal(want), "desc=%v", desc)
	}
}
