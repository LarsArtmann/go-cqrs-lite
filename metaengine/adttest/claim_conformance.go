// Conformance suites for the ADR-0142 write-side capabilities:
// [AssertDueClaimer] and [AssertDedupStore].

package adttest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// claimConformanceCollection returns a run-unique collection name so repeated
// runs (reruns, -count>1) never observe state from a previous run on
// persistent engines (same suffix trick as Scenarios()).
func claimConformanceCollection(base string) string {
	return fmt.Sprintf("%s_claims_%d", base, time.Now().UnixNano())
}

// AssertDueClaimer runs the DueClaimer conformance suite against every
// factory. An engine that does NOT implement metaengine.DueClaimer FAILS (no
// silent skip — unlike RunMatrix, calling this asserts the capability).
// Run under -race for the exclusivity guarantees.
//
//	func TestEngineDueClaims(t *testing.T) {
//	    adttest.AssertDueClaimer(t, []adttest.Factory{
//	        {Name: "sqlite", Create: func(t *testing.T) metaengine.Engine { return newSQLiteEngine(t) }},
//	    })
//	}
func AssertDueClaimer( //nolint:maintidx // linear conformance narrative
	t *testing.T,
	factories []Factory,
) {
	t.Helper()

	if len(factories) == 0 {
		t.Fatal("AssertDueClaimer requires at least 1 factory")
	}

	for _, factory := range factories {
		t.Run(factory.Name, func(t *testing.T) {
			eng := factory.Create(t)
			defer metaengine.DeferClose(eng)

			claimer, ok := eng.(metaengine.DueClaimer)
			if !ok {
				t.Fatalf("%s does not implement metaengine.DueClaimer", factory.Name)
			}

			ctx := context.Background()

			t.Run("ClaimInsertIdempotent", func(t *testing.T) {
				col := claimConformanceCollection("idem")
				mustClaimT(t, claimer.ClaimInsert(ctx, col, "t1", ms(100), []byte("first")))
				mustClaimT(t, claimer.ClaimInsert(ctx, col, "t1", ms(200), []byte("second")))

				claims := claimDueT(t, ctx, claimer, col, ms(1000))
				if len(claims) != 1 || string(claims[0].Payload) != "first" {
					t.Fatalf("idempotency violated: %+v", claims)
				}
			})

			t.Run("NotBeforeGatingAndDueOrdering", func(t *testing.T) {
				col := claimConformanceCollection("order")
				now := ms(1000)

				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "late", now.Add(time.Hour), []byte("l")),
				)
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "b", now.Add(-2*time.Second), []byte("2")),
				)
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "a", now.Add(-2*time.Second), []byte("1")),
				)
				mustClaimT(t, claimer.ClaimInsert(ctx, col, "c", now.Add(-time.Hour), []byte("0")))

				claims := claimDueT(t, ctx, claimer, col, now)
				want := []string{"c", "a", "b"}
				if len(claims) != len(want) {
					t.Fatalf(
						"got %d claims, want %d (NotBefore must gate 'late')",
						len(claims),
						len(want),
					)
				}

				order := make([]string, 0, len(claims))
				for _, cl := range claims {
					order = append(order, cl.Key)
				}

				for i, key := range want {
					if order[i] != key {
						t.Fatalf("order = %v, want %v (DueAt asc, key tie-break)", order, want)
					}
				}
			})

			t.Run("LeaseFenceAndExpiryReclaim", func(t *testing.T) {
				col := claimConformanceCollection("fence")
				now := ms(1000)

				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "only", now.Add(-time.Minute), []byte("w")),
				)

				first := claimDueT(t, ctx, claimer, col, now)
				if len(first) != 1 {
					t.Fatalf("first claim got %d, want 1", len(first))
				}

				if second := claimDueT(t, ctx, claimer, col, now); len(second) != 0 {
					t.Fatalf("lease fence violated: %+v", second)
				}

				reclaimed := claimDueT(t, ctx, claimer, col, now.Add(61*time.Second))
				if len(reclaimed) != 1 || reclaimed[0].Key != "only" {
					t.Fatalf("lease-expiry reclaim failed: %+v", reclaimed)
				}
			})

			t.Run("RenewLeaseOwnerFenced", func(t *testing.T) {
				col := claimConformanceCollection("renew")
				now := ms(1000)

				mustClaimT(t, claimer.ClaimInsert(ctx, col, "k", now.Add(-time.Minute), nil))

				owned := claimDueT(t, ctx, claimer, col, now)
				if len(owned) != 1 {
					t.Fatalf("claim: %d, want 1", len(owned))
				}

				if err := claimer.RenewLease(
					ctx,
					col,
					"k",
					"w1",
					time.Minute,
					now.Add(30*time.Second),
				); err != nil {
					t.Fatalf("owner renew: %v", err)
				}

				if err := claimer.RenewLease(
					ctx,
					col,
					"k",
					"w2",
					time.Minute,
					now.Add(30*time.Second),
				); !errors.Is(
					err,
					metaengine.ErrClaimLeaseNotHeld,
				) {
					t.Fatalf("foreign renew: want ErrClaimLeaseNotHeld, got %v", err)
				}
			})

			t.Run("ClaimDeleteIdempotent", func(t *testing.T) {
				col := claimConformanceCollection("del")

				mustClaimT(t, claimer.ClaimDelete(ctx, col, "absent")) // absent delete is success
				mustClaimT(t, claimer.ClaimInsert(ctx, col, "k", ms(1), nil))
				mustClaimT(t, claimer.ClaimDelete(ctx, col, "k"))
				mustClaimT(t, claimer.ClaimDelete(ctx, col, "k"))

				if claims := claimDueT(t, ctx, claimer, col, ms(10_000)); len(claims) != 0 {
					t.Fatalf("deleted item still claimable: %+v", claims)
				}
			})

			t.Run("DeleteIfDueEpochGuard", func(t *testing.T) {
				col := claimConformanceCollection("epoch")
				now := ms(1000)

				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "t", now.Add(-time.Minute), []byte("gen1")),
				)

				claims := claimDueT(t, ctx, claimer, col, now)
				if len(claims) != 1 {
					t.Fatalf("claim: %d", len(claims))
				}

				// Re-schedule under the same key; a stale epoch delete must
				// not remove generation 2.
				mustClaimT(t, claimer.ClaimDelete(ctx, col, "t"))
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "t", now.Add(time.Hour), []byte("gen2")),
				)
				mustClaimT(t, claimer.ClaimDeleteIfDue(ctx, col, "t", claims[0].DueAt))

				after := claimDueT(t, ctx, claimer, col, now.Add(2*time.Hour))
				if len(after) != 1 || string(after[0].Payload) != "gen2" {
					t.Fatalf("stale finalizer deleted the re-scheduled generation: %+v", after)
				}
			})

			t.Run("ClaimLimit", func(t *testing.T) {
				col := claimConformanceCollection("limit")
				now := ms(1000)

				for i := range 10 {
					mustClaimT(
						t,
						claimer.ClaimInsert(
							ctx,
							col,
							fmt.Sprintf("k%02d", i),
							now.Add(-time.Minute),
							nil,
						),
					)
				}

				claims, err := claimer.ClaimDue(ctx, metaengine.ClaimDueRequest{
					Collection: col, Owner: "w", Lease: time.Minute, Limit: 3, Now: now,
				})
				if err != nil {
					t.Fatalf("ClaimDue limit: %v", err)
				}

				if len(claims) != 3 {
					t.Fatalf("limit: got %d claims, want 3", len(claims))
				}
			})

			t.Run("ConcurrentClaimersDisjoint", func(t *testing.T) {
				col := claimConformanceCollection("race")
				now := ms(1000)

				const items = 40

				for i := range items {
					mustClaimT(
						t,
						claimer.ClaimInsert(
							ctx,
							col,
							fmt.Sprintf("k%03d", i),
							now.Add(-time.Minute),
							nil,
						),
					)
				}

				var (
					wg      sync.WaitGroup
					mu      sync.Mutex
					allKeys = map[string]int{}
				)

				for worker := range 4 {
					wg.Add(1)

					go func() {
						defer wg.Done()

						claimed, err := claimer.ClaimDue(ctx, metaengine.ClaimDueRequest{
							Collection: col,
							Owner:      fmt.Sprintf("w%d", worker),
							Lease:      time.Minute,
							Now:        now,
						})
						if err != nil {
							t.Errorf("concurrent claim: %v", err)

							return
						}

						mu.Lock()
						defer mu.Unlock()

						for _, cl := range claimed {
							allKeys[cl.Key]++
						}
					}()
				}

				wg.Wait()

				if len(allKeys) != items {
					t.Fatalf("claimed %d distinct keys, want %d", len(allKeys), items)
				}

				for key, count := range allKeys {
					if count != 1 {
						t.Fatalf("key %s claimed %d times — fence violated", key, count)
					}
				}
			})
		})
	}
}
