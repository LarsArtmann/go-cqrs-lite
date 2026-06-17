package id_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/onsi/gomega"
)

func TestMonotonicOrderingWithinSameMillisecond(t *testing.T) {
	g := gomega.NewWithT(t)

	// Generate 100 IDs rapidly — they should be monotonically ordered
	// since they're likely within the same millisecond.
	const count = 100 //nolint:mnd // test count
	ids := make([]id.EventID, count)

	for i := range ids {
		ids[i] = id.NewEventID()
	}

	// IDs generated within the same ms should be monotonically increasing.
	// We can't guarantee they're in the same ms, but monotonic entropy ensures
	// they're always ordered even across ms boundaries.
	for i := 1; i < count; i++ {
		g.Expect(ids[i].Get().Compare(ids[i-1].Get())).
			To(gomega.BeNumerically(">", 0), "ID %d should be > ID %d", i, i-1)
	}
}

func TestConcurrentGenerationIsSafe(t *testing.T) {
	const goroutines = 50 //nolint:mnd // test count
	const perGoroutine = 20

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for g := range goroutines {
		go func(_ int) {
			defer wg.Done()

			for range perGoroutine {
				_ = id.NewEventID()
			}
		}(g)
	}

	wg.Wait()
	// If we get here without panic, the monotonic source is thread-safe.
}

func TestAllIDsAreUnique(t *testing.T) {
	g := gomega.NewWithT(t)

	const count = 1000 //nolint:mnd // test count
	seen := make(map[string]bool, count)

	for range count {
		idVal := id.NewEventID()
		s := idVal.String()
		g.Expect(seen[s]).To(gomega.BeFalse(), "duplicate ID generated")
		seen[s] = true
	}
}
