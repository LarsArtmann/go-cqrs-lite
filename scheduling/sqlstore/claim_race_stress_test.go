package sqlstore_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sqlstore "github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestClaimingSQLite_RaceStress_DueVsMetrics runs concurrent Due pollers
// against one store while a Metrics reader hammers the counters (W2.10a).
// Under -race this catches unsynchronized counter access; two semantic
// invariants ride along: every scheduled timer is claimed EXACTLY once
// across all pollers (claim = ownership transfer, never a double delivery),
// and the built-in counters agree with what the pollers actually observed.
func TestClaimingSQLite_RaceStress_DueVsMetrics(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	const totalTimers = 60

	for i := range totalTimers {
		if err := store.Schedule(ctx, scheduling.Timer[struct{}]{
			ID:     scheduling.MustParseTimerID(fmt.Sprintf("race-%03d", i)),
			FireAt: now.Add(-time.Second),
		}); err != nil {
			t.Fatalf("Schedule %d: %v", i, err)
		}
	}

	var claimedTotal atomic.Int64

	var mu sync.Mutex

	var claimedIDs []string

	const pollers = 4

	const pollsPerPoller = 25

	var wg sync.WaitGroup

	stop := make(chan struct{})

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
			}

			_ = store.Metrics()
		}
	}()

	errCh := make(chan error, pollers)

	for range pollers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range pollsPerPoller {
				claimed, err := store.Due(ctx, time.Now().UTC())
				if err != nil {
					errCh <- err

					return
				}

				claimedTotal.Add(int64(len(claimed)))

				mu.Lock()

				for _, timer := range claimed {
					claimedIDs = append(claimedIDs, timer.ID.String())
				}

				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	close(stop)

	close(errCh)

	for err := range errCh {
		t.Fatalf("Due poller: %v", err)
	}

	slices.Sort(claimedIDs)

	if n := len(claimedIDs); n != totalTimers {
		t.Errorf("claimed %d timers across %d pollers, want exactly %d (each timer claimed once)", n, pollers, totalTimers)
	}

	for i := 1; i < len(claimedIDs); i++ {
		if claimedIDs[i] == claimedIDs[i-1] {
			t.Errorf("timer %s claimed more than once — double delivery", claimedIDs[i])

			break
		}
	}

	metrics := store.Metrics()

	if metrics.ClaimedTimers != claimedTotal.Load() {
		t.Errorf(
			"Metrics.ClaimedTimers = %d, pollers observed %d — counters and claims diverged",
			metrics.ClaimedTimers, claimedTotal.Load(),
		)
	}
}
