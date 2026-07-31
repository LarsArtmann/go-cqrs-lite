package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPebbleLayoutPlanner_EmptyFilterResults verifies that a filter matching
// zero items returns an empty slice, not nil, and doesn't panic.
func TestPebbleLayoutPlanner_EmptyFilterResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"status"}, nil)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)
	gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", "k1", map[string]any{"status": "active"})).To(gomega.Succeed())

	rsr := eng.(metaengine.RawScanReader)

	results, err := rsr.ScanRawValues(ctx, "items",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "nonexistent"}},
		nil, nil, 0,
	)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results).To(gomega.BeEmpty())
}

// TestPebbleLayoutPlanner_ConcurrentReadWrite verifies that concurrent writes
// and scans don't race or corrupt data.
func TestPebbleLayoutPlanner_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"score"}, nil)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	const writers = 4
	const itemsPerWriter = 25

	var wg sync.WaitGroup

	for w := range writers {
		wg.Add(1)

		go func(writerID int) {
			defer wg.Done()

			for i := range itemsPerWriter {
				key := fmt.Sprintf("w%d-i%d", writerID, i)
				_ = mb.MapSet(ctx, "items", key, map[string]any{
					"score": float64(writerID*100 + i),
				})
			}
		}(w)
	}

	for range 3 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = rsr.ScanRawValues(ctx, "items",
				[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: 0}},
				&metaengine.SortSpec{Column: "score"}, nil, 10,
			)
		}()
	}

	wg.Wait()

	results, err := rsr.ScanRawValues(ctx, "items", nil, nil, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results).To(gomega.HaveLen(writers * itemsPerWriter))
}

// TestPebbleLayoutPlanner_KeyCollision verifies that writing the same key
// twice (update) doesn't create duplicate index entries.
func TestPebbleLayoutPlanner_KeyCollision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"score"}, nil)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	// Write the same key 5 times with different scores.
	for i := 1; i <= 5; i++ {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", "k1", map[string]any{
			"score": float64(i * 10),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)

	results, err := rsr.ScanRawValues(ctx, "items", nil, nil, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results).To(gomega.HaveLen(1), "key collision: expected 1 result, got %d", len(results))

	var decoded map[string]any
	gomega.NewWithT(t).Expect(json.Unmarshal(results[0], &decoded)).To(gomega.Succeed())
	gomega.NewWithT(t).Expect(decoded["score"]).To(gomega.Equal(float64(50)), "expected last write to win")
}

// TestPebbleLayoutPlanner_NoLayoutFullScan verifies that ScanRawValues works
// without a declared layout (falls back to full scan with Go-level filtering).
func TestPebbleLayoutPlanner_NoLayoutFullScan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for i := 1; i <= 5; i++ {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"score": float64(i * 10),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "score", Desc: true}

	// Full scan with filter + sort (no layout plan).
	results, err := rsr.ScanRawValues(ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGt, Value: 20}},
		sortSpec, nil, 0,
	)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results).To(gomega.HaveLen(3))

	scores := extractField[float64](t, results, "score")
	gomega.NewWithT(t).Expect(scores).To(gomega.Equal([]float64{50, 40, 30}))
}
