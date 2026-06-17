package id_test

import (
	"sync"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

const (
	testIDCount       = 100
	testGoroutines    = 50
	testPerGoroutine  = 20
	testUniqueIDCount = 1000
)

func TestMonotonicOrderingWithinSameMillisecond(t *testing.T) {
	g := gomega.NewWithT(t)

	ids := make([]id.EventID, testIDCount)

	for i := range ids {
		ids[i] = id.NewEventID()
	}

	for i := 1; i < testIDCount; i++ {
		g.Expect(ids[i].Get().Compare(ids[i-1].Get())).
			To(gomega.BeNumerically(">", 0), "ID %d should be > ID %d", i, i-1)
	}
}

func TestConcurrentGenerationIsSafe(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(testGoroutines)

	for range testGoroutines {
		go func() {
			defer wg.Done()

			for range testPerGoroutine {
				_ = id.NewEventID()
			}
		}()
	}

	wg.Wait()
}

func TestAllIDsAreUnique(t *testing.T) {
	g := gomega.NewWithT(t)

	seen := make(map[string]bool, testUniqueIDCount)

	for range testUniqueIDCount {
		idVal := id.NewEventID()
		s := idVal.String()
		g.Expect(seen[s]).To(gomega.BeFalse(), "duplicate ID generated")
		seen[s] = true
	}
}
