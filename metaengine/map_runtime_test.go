package metaengine_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mapRuntimeHosts returns engine factories exercising both stored-value
// shapes: the memory engine (typed struct round-trip) and the SQLite engine
// (JSON map[string]any round-trip through reify).
func mapRuntimeHosts(t *testing.T) map[string]metaengine.Engine {
	t.Helper()

	hosts := map[string]metaengine.Engine{"memory": metaengine.NewMemoryEngine()}
	t.Cleanup(func() { _ = hosts["memory"].Close() })

	// A UNIQUE named in-memory database: plain file::memory:?cache=shared is
	// one database per PROCESS, so parallel test instances (-count>1) would
	// close it out from under each other and invalidate cached statements.
	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:adttest_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1) // one connection: the named memory db lives while it exists

	t.Cleanup(func() { _ = db.Close() })

	sq, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = sq.Close() })

	hosts["sqlite"] = sq

	return hosts
}

func newClaimer(t *testing.T, eng metaengine.Engine) *metaengine.MapDueClaimer {
	t.Helper()

	c, err := metaengine.NewMapDueClaimer(eng)
	if err != nil {
		t.Fatalf("NewMapDueClaimer: %v", err)
	}

	return c
}

func newDedup(t *testing.T, eng metaengine.Engine) *metaengine.MapDedupStore {
	t.Helper()

	d, err := metaengine.NewMapDedupStore(eng)
	if err != nil {
		t.Fatalf("NewMapDedupStore: %v", err)
	}

	return d
}

func TestMapDueClaimer_ClaimInsertIdempotent(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClaimer(t, eng)
			ctx := context.Background()

			must(t, c.ClaimInsert(ctx, "timers", "t1", time.UnixMilli(100), []byte("first")))
			must(t, c.ClaimInsert(ctx, "timers", "t1", time.UnixMilli(200), []byte("second")))

			claims, err := c.ClaimDue(
				ctx,
				metaengine.ClaimDueRequest{Collection: "timers", Now: time.UnixMilli(1000)},
			)
			must(t, err)

			if len(claims) != 1 || string(claims[0].Payload) != "first" {
				t.Fatalf("idempotency violated: %+v", claims)
			}

			if !claims[0].DueAt.Equal(time.UnixMilli(100)) {
				t.Fatalf("dueAt changed by re-insert: %v", claims[0].DueAt)
			}
		})
	}
}

func TestMapDueClaimer_NotBeforeAndOrdering(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClaimer(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			must(t, c.ClaimInsert(ctx, "timers", "late", now.Add(time.Hour), []byte("l")))
			must(t, c.ClaimInsert(ctx, "timers", "b-second", now.Add(-2*time.Second), []byte("2")))
			must(t, c.ClaimInsert(ctx, "timers", "a-first", now.Add(-2*time.Second), []byte("1")))
			must(t, c.ClaimInsert(ctx, "timers", "c-earlier", now.Add(-time.Hour), []byte("0")))

			claims, err := c.ClaimDue(
				ctx,
				metaengine.ClaimDueRequest{Collection: "timers", Now: now},
			)
			must(t, err)

			want := []string{"c-earlier", "a-first", "b-second"}
			if len(claims) != len(want) {
				t.Fatalf("got %d claims, want %d (late must be gated)", len(claims), len(want))
			}

			for i, key := range want {
				if claims[i].Key != key {
					t.Fatalf(
						"order[%d] = %s, want %s (DueAt asc, key tie-break)",
						i,
						claims[i].Key,
						key,
					)
				}
			}
		})
	}
}

func TestMapDueClaimer_ExclusivityAndReclaim(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClaimer(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			must(t, c.ClaimInsert(ctx, "tasks", "only", now.Add(-time.Minute), []byte("work")))

			first, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "tasks", Owner: "w1", Lease: time.Minute, Now: now,
			})
			must(t, err)

			if len(first) != 1 {
				t.Fatalf("first claim got %d, want 1", len(first))
			}

			second, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "tasks", Owner: "w2", Lease: time.Minute, Now: now,
			})
			must(t, err)

			if len(second) != 0 {
				t.Fatalf("lease fence violated: w2 claimed %v", second)
			}

			reclaimed, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "tasks",
				Owner:      "w2",
				Lease:      time.Minute,
				Now:        now.Add(61 * time.Second),
			})
			must(t, err)

			if len(reclaimed) != 1 || reclaimed[0].Key != "only" {
				t.Fatalf("expiry reclaim failed: %+v", reclaimed)
			}
		})
	}
}

func TestMapDueClaimer_RenewLease(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClaimer(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			must(t, c.ClaimInsert(ctx, "tasks", "k", now.Add(-time.Minute), nil))

			_, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "tasks", Owner: "w1", Lease: time.Minute, Now: now,
			})
			must(t, err)

			must(t, c.RenewLease(ctx, "tasks", "k", "w1", time.Minute, now.Add(30*time.Second)))

			err = c.RenewLease(ctx, "tasks", "k", "w2", time.Minute, now.Add(30*time.Second))
			if !errors.Is(err, metaengine.ErrClaimLeaseNotHeld) {
				t.Fatalf("foreign renew: want ErrClaimLeaseNotHeld, got %v", err)
			}

			err = c.RenewLease(ctx, "tasks", "k", "w1", time.Minute, now.Add(2*time.Minute))
			if !errors.Is(err, metaengine.ErrClaimLeaseNotHeld) {
				t.Fatalf("lapsed renew: want ErrClaimLeaseNotHeld, got %v", err)
			}
		})
	}
}

func TestMapDueClaimer_DeleteIfDueEpochGuard(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClaimer(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			must(t, c.ClaimInsert(ctx, "timers", "t", now.Add(-time.Minute), []byte("gen1")))

			claims, err := c.ClaimDue(
				ctx,
				metaengine.ClaimDueRequest{Collection: "timers", Now: now},
			)
			must(t, err)

			if len(claims) != 1 {
				t.Fatalf("claim: %d", len(claims))
			}

			gen1Due := claims[0].DueAt

			// Re-schedule the same key with a new epoch while the old
			// dispatch is still in flight; a stale ClaimDeleteIfDue(gen1)
			// must NOT kill generation 2 (the MarkFired race fix).
			must(t, c.ClaimDelete(ctx, "timers", "t"))
			must(t, c.ClaimInsert(ctx, "timers", "t", now.Add(time.Hour), []byte("gen2")))
			must(t, c.ClaimDeleteIfDue(ctx, "timers", "t", gen1Due))

			after, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "timers", Now: now.Add(2 * time.Hour),
			})
			must(t, err)

			if len(after) != 1 || string(after[0].Payload) != "gen2" {
				t.Fatalf("stale finalizer deleted the re-scheduled generation: %+v", after)
			}
		})
	}
}

func TestMapDueClaimer_ConcurrentClaimersDisjoint(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	c := newClaimer(t, eng)
	ctx := context.Background()

	const items = 40

	for i := range items {
		must(t, c.ClaimInsert(ctx, "tasks", keyN(i), time.UnixMilli(1), []byte("v")))
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

			claimed, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{
				Collection: "tasks",
				Owner:      keyN(worker),
				Lease:      time.Minute,
				Now:        time.UnixMilli(1000),
			})
			if err != nil {
				t.Errorf("claim: %v", err)

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
}

func TestMapDedupStore_CheckAndRecordAndExpiry(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d := newDedup(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			seen, err := d.DedupCheckAndRecord(ctx, "dedup", "cmd-1", time.Minute, now)
			must(t, err)

			if seen {
				t.Fatal("first CheckAndRecord must report unseen")
			}

			seen, err = d.DedupCheckAndRecord(
				ctx,
				"dedup",
				"cmd-1",
				time.Minute,
				now.Add(time.Second),
			)
			must(t, err)

			if !seen {
				t.Fatal("duplicate CheckAndRecord must report seen")
			}

			seen, err = d.DedupSeen(ctx, "dedup", "cmd-1", now.Add(time.Second))
			must(t, err)

			if !seen {
				t.Fatal("Seen must be true inside the window")
			}

			after := now.Add(2 * time.Minute)

			seen, err = d.DedupSeen(ctx, "dedup", "cmd-1", after)
			must(t, err)

			if seen {
				t.Fatal("expired key must read unseen")
			}

			seen, err = d.DedupCheckAndRecord(ctx, "dedup", "cmd-1", time.Minute, after)
			must(t, err)

			if seen {
				t.Fatal("expired window must be re-claimable")
			}
		})
	}
}

func TestMapDedupStore_Sweep(t *testing.T) {
	t.Parallel()

	for name, eng := range mapRuntimeHosts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d := newDedup(t, eng)
			ctx := context.Background()
			now := time.UnixMilli(1000)

			for i := range 5 {
				_, err := d.DedupCheckAndRecord(ctx, "dedup", keyN(i), time.Minute, now)
				must(t, err)
			}

			_, err := d.DedupCheckAndRecord(ctx, "dedup", "fresh", time.Hour, now)
			must(t, err)

			removed, err := d.DedupSweep(ctx, "dedup", now.Add(2*time.Minute))
			must(t, err)

			if removed != 5 {
				t.Fatalf(
					"sweep removed %d, want 5 (the 1-minute windows; 'fresh' has an hour)",
					removed,
				)
			}

			seen, err := d.DedupSeen(ctx, "dedup", "fresh", now.Add(2*time.Minute))
			must(t, err)

			if !seen {
				t.Fatal("sweep must not remove the still-live 'fresh' window")
			}
		})
	}
}

func TestMapDedupStore_ConcurrentCASExactlyOneWinner(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	d := newDedup(t, eng)
	ctx := context.Background()

	const racers = 16

	var wg sync.WaitGroup

	now := time.UnixMilli(1000)

	trues := make(chan bool, racers)

	for range racers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			seen, err := d.DedupCheckAndRecord(ctx, "dedup", "race", time.Minute, now)
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
		t.Fatalf("CAS violated: %d racers saw seen, want exactly %d", seenCount, racers-1)
	}
}

func keyN(i int) string { return fmt.Sprintf("k%03d", i) }

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
