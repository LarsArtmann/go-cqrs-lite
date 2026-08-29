package metaengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type (
	raceEvt     struct{ ID, Val string }
	raceQueryIn struct{ Limit int }
	raceResult  struct{ Items []string }
)

// TestOnRecordFolds_ConcurrentStoresNoRace pins the recHolder fix: two Stores
// planned from the same package-level declarations share fold instances, and
// each Store's fold locks only serialize within its own Store. Concurrent
// Apply on both stores exercised the shared record cell as unsynchronized
// mutable state (data race + cross-attribution). Run with -race.
func TestOnRecordFolds_ConcurrentStoresNoRace(t *testing.T) {
	decl := metaengine.Query[raceQueryIn, raceResult]("race_q",
		metaengine.OnRecord(raceEvt{}, func(rec record.Record, e raceEvt) (string, string) {
			return e.ID, rec.Type + "|" + e.Val
		}),
	)

	newStore := func() *metaengine.Store {
		s, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, decl)
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}

		t.Cleanup(func() { _ = s.Close() })

		return s
	}

	live := newStore()
	replay := newStore()

	ctx := context.Background()

	var wg sync.WaitGroup

	for _, s := range []*metaengine.Store{live, replay} {
		wg.Add(1)

		go func(s *metaengine.Store) {
			defer wg.Done()

			for i := range 300 {
				evt := raceEvt{ID: fmt.Sprintf("e%d", i), Val: "v"}

				if err := s.Apply(ctx, "raceEvt", evt); err != nil {
					t.Errorf("apply %d: %v", i, err)

					return
				}
			}
		}(s)
	}

	wg.Wait()
}
