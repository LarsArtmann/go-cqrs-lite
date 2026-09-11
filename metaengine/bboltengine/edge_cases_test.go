package bboltengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
)

// TestBboltEdgeCases_EmptyFilterResults verifies that a MapScan with a filter
// matching zero items returns an empty result, not nil, and doesn't panic.
func TestBboltEdgeCases_EmptyFilterResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := mustNewBboltEngine(t)

	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.ScanBackend)

	g := gomega.NewWithT(t)
	g.Expect(mb.MapSet(ctx, "items", "k1", map[string]any{"status": "active"})).To(gomega.Succeed())

	results, err := sb.MapScan(ctx, "items",
		func(item any) bool { return false }, // filter nothing
		nil, nil, 0,
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.BeEmpty())
}

// TestBboltEdgeCases_ConcurrentWrites verifies that concurrent writes to
// different keys don't race or corrupt data. bbolt is single-writer, so
// writes are serialized; this test ensures no panics or deadlocks.
func TestBboltEdgeCases_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := mustNewBboltEngine(t)

	mb := eng.(metaengine.MapBackend)

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

	wg.Wait()

	sb := eng.(metaengine.ScanBackend)
	results, err := sb.MapScan(ctx, "items", nil, nil, nil, 0)
	g := gomega.NewWithT(t)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.HaveLen(writers * itemsPerWriter))
}

// TestBboltEdgeCases_KeyCollision verifies that writing the same key
// twice (update) doesn't create duplicate entries.
func TestBboltEdgeCases_KeyCollision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := mustNewBboltEngine(t)

	mb := eng.(metaengine.MapBackend)

	for i := 1; i <= 5; i++ {
		g := gomega.NewWithT(t)
		g.Expect(mb.MapSet(ctx, "items", "k1", map[string]any{
			"score": float64(i * 10),
		})).To(gomega.Succeed())
	}

	sb := eng.(metaengine.ScanBackend)
	results, err := sb.MapScan(ctx, "items", nil, nil, nil, 0)
	g := gomega.NewWithT(t)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(results.Items).To(gomega.HaveLen(1))

	// bbolt's MapScan returns decoded values (map[string]any).
	if m, ok := results.Items[0].(map[string]any); ok {
		g.Expect(m["score"]).To(gomega.Equal(float64(50)))
	}
}

// TestBboltEdgeCases_LargeValue verifies that large JSON values can be
// stored and retrieved correctly.
func TestBboltEdgeCases_LargeValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := mustNewBboltEngine(t)

	mb := eng.(metaengine.MapBackend)
	g := gomega.NewWithT(t)

	largeData := make(map[string]any)
	for i := range 100 {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	g.Expect(mb.MapSet(ctx, "large", "big", largeData)).To(gomega.Succeed())

	val, found, err := mb.MapGet(ctx, "large", "big")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())

	result, ok := val.(map[string]any)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(result).To(gomega.HaveKey("field_50"))
}
