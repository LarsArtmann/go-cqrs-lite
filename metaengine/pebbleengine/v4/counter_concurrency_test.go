package pebbleengine_test

import (
	"sync"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPebbleCounterIncrement_ConcurrentIsNotLost runs many parallel
// increments of one counter through the CounterBackend facet; every one must
// be accounted for.
func TestPebbleCounterIncrement_ConcurrentIsNotLost(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	cb := eng.(metaengine.CounterBackend)

	const goroutines, perGoroutine = 24, 25

	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range perGoroutine {
				if err := cb.CounterIncrement(t.Context(), "counters", metaengine.Delta{"c": 1}); err != nil {
					t.Errorf("CounterIncrement: %v", err)

					return
				}
			}
		}()
	}

	wg.Wait()

	got, err := cb.CounterGet(t.Context(), "counters")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(got).To(gomega.HaveKeyWithValue("c", int64(goroutines*perGoroutine)))
}
