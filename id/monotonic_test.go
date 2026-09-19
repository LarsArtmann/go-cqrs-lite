package id_test

import (
	"encoding/binary"
	"sync"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

// TestParallelGenerationUniquePerMillisecond stress-tests the lock-free
// generator across goroutines: every ID is globally unique, and all IDs
// sharing a millisecond share that millisecond's random prefix with
// collision-free counter suffixes (the single-epoch-per-ms invariant).
func TestParallelGenerationUniquePerMillisecond(t *testing.T) {
	g := gomega.NewWithT(t)

	const (
		goroutines = 8
		perG       = 4096
	)
	total := goroutines * perG

	results := make([]id.EventID, total)
	var wg sync.WaitGroup

	wg.Add(goroutines)

	for g := range goroutines {
		go func(slot int) {
			defer wg.Done()

			for i := range perG {
				results[slot*perG+i] = id.NewEventID()
			}
		}(g)
	}

	wg.Wait()

	seen := make(map[string]bool, total)
	perMsSuffix := make(map[uint64]map[uint64]bool)
	perMsPrefix := make(map[uint64]string)

	for _, idVal := range results {
		s := idVal.String()
		g.Expect(seen[s]).To(gomega.BeFalse(), "duplicate ID generated under parallelism")
		seen[s] = true

		u := idVal.Get()
		ms := binary.BigEndian.Uint64(u[0:8]) >> 16
		prefix := string(u[6:12])
		suffix := uint64(binary.BigEndian.Uint32(u[12:16]))

		if prev, ok := perMsPrefix[ms]; ok {
			g.Expect(prefix).To(gomega.Equal(prev), "two prefixes within millisecond %d", ms)
		} else {
			perMsPrefix[ms] = prefix
		}

		if perMsSuffix[ms] == nil {
			perMsSuffix[ms] = make(map[uint64]bool)
		}

		g.Expect(perMsSuffix[ms][suffix]).
			To(gomega.BeFalse(), "duplicate suffix %d within millisecond %d", suffix, ms)
		perMsSuffix[ms][suffix] = true
	}
}
