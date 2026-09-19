package adttest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertDedupStore runs the DedupStore conformance suite against every
// factory; an engine without the capability FAILS. Run under -race.
func AssertDedupStore(t *testing.T, factories []Factory) {
	t.Helper()

	if len(factories) == 0 {
		t.Fatal("AssertDedupStore requires at least 1 factory")
	}

	for _, factory := range factories {
		t.Run(factory.Name, func(t *testing.T) {
			eng := factory.Create(t)
			defer metaengine.DeferClose(eng)

			dedup, ok := eng.(metaengine.DedupStore)
			if !ok {
				t.Fatalf("%s does not implement metaengine.DedupStore", factory.Name)
			}

			ctx := context.Background()
			col := fmt.Sprintf("dedup_conf_%d", time.Now().UnixNano())
			now := ms(1000)

			t.Run("CASWindow", func(t *testing.T) {
				seen, err := dedup.DedupCheckAndRecord(ctx, col, "cmd-1", time.Minute, now)
				mustClaimT(t, err)

				if seen {
					t.Fatal("first CheckAndRecord must report unseen")
				}

				seen, err = dedup.DedupCheckAndRecord(
					ctx,
					col,
					"cmd-1",
					time.Minute,
					now.Add(time.Second),
				)
				mustClaimT(t, err)

				if !seen {
					t.Fatal("duplicate CheckAndRecord must report seen")
				}

				seen, err = dedup.DedupSeen(ctx, col, "cmd-1", now.Add(time.Second))
				mustClaimT(t, err)

				if !seen {
					t.Fatal("Seen must be true inside the window")
				}
			})

			t.Run("TTLExpiryReclaim", func(t *testing.T) {
				after := now.Add(2 * time.Minute)

				seen, err := dedup.DedupSeen(ctx, col, "cmd-1", after)
				mustClaimT(t, err)

				if seen {
					t.Fatal("expired key must read unseen")
				}

				seen, err = dedup.DedupCheckAndRecord(ctx, col, "cmd-1", time.Minute, after)
				mustClaimT(t, err)

				if seen {
					t.Fatal("expired window must be re-claimable")
				}
			})

			t.Run("Sweep", func(t *testing.T) {
				sweepCol := fmt.Sprintf("dedup_sweep_%d", time.Now().UnixNano())

				for i := range 5 {
					_, err := dedup.DedupCheckAndRecord(
						ctx,
						sweepCol,
						fmt.Sprintf("k%d", i),
						time.Minute,
						now,
					)
					mustClaimT(t, err)
				}

				_, err := dedup.DedupCheckAndRecord(ctx, sweepCol, "fresh", time.Hour, now)
				mustClaimT(t, err)

				removed, err := dedup.DedupSweep(ctx, sweepCol, now.Add(2*time.Minute))
				mustClaimT(t, err)

				if removed != 5 {
					t.Fatalf("sweep removed %d, want 5 (the lapsed windows only)", removed)
				}

				seen, err := dedup.DedupSeen(ctx, sweepCol, "fresh", now.Add(2*time.Minute))
				mustClaimT(t, err)

				if !seen {
					t.Fatal("sweep must not remove still-live windows")
				}
			})

			t.Run("ConcurrentCASExactlyOneWinner", func(t *testing.T) {
				racers := casRacers(t)

				var wg sync.WaitGroup

				trues := make(chan bool, racers)

				for range racers {
					wg.Add(1)

					go func() {
						defer wg.Done()

						seen, err := dedup.DedupCheckAndRecord(ctx, col, "race", time.Minute, now)
						if err != nil {
							t.Errorf("racer: %v", err)

							return
						}

						trues <- seen
					}()
				}

				wg.Wait()
				close(trues)

				seenCount := 0

				for seen := range trues {
					if seen {
						seenCount++
					}
				}

				if seenCount != racers-1 {
					t.Fatalf(
						"CAS violated: %d racers saw seen, want exactly %d",
						seenCount,
						racers-1,
					)
				}
			})
		})
	}
}

// casRacers reports how many concurrent CAS racers
// ConcurrentCASExactlyOneWinner spawns. Default 16; ADTTEST_CAS_RACERS (>= 2)
// caps it for constrained runners — QEMU's slirp networking resets bursts of
// concurrent MySQL connections before the server ever sees them (an infra
// limit, not a store-semantics limit).
func casRacers(t *testing.T) int {
	t.Helper()

	if v, err := strconv.Atoi(os.Getenv("ADTTEST_CAS_RACERS")); err == nil && v >= 2 {
		return v
	}

	return 16
}

func claimDueT(
	t *testing.T,
	ctx context.Context,
	claimer metaengine.DueClaimer,
	col string,
	now time.Time,
) []metaengine.DueClaim {
	t.Helper()

	claims, err := claimer.ClaimDue(ctx, metaengine.ClaimDueRequest{
		Collection: col, Owner: "w1", Now: now,
	})
	if err != nil {
		t.Fatalf("ClaimDue %s: %v", col, err)
	}

	return claims
}

func mustClaimT(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func ms(msec int64) time.Time { return time.UnixMilli(msec) }
