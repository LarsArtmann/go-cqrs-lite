package engine_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

type payload struct {
	OrderID string `json:"orderId"`
}

func newStore(t *testing.T) *engine.TimerStore[payload] {
	t.Helper()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:schedengine_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	store, err := engine.NewTimerStore[payload](eng)
	if err != nil {
		t.Fatalf("NewTimerStore: %v", err)
	}

	return store
}

func timer(id string, fireAt time.Time, order string) scheduling.Timer[payload] {
	return scheduling.Timer[payload]{
		ID:     scheduling.MustParseTimerID(id),
		FireAt: fireAt,
		Payload: payload{
			OrderID: order,
		},
	}
}

// TestTimerStore_Parity pins behavioral parity with the TimerStore contract
// (and scheduling.MemoryTimerStore): idempotent Schedule, Due ordering and
// gating, fire-once exclusivity, MarkFired, Cancel.
func TestTimerStore_Parity(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()
	now := time.UnixMilli(1000)

	if err := store.Schedule(ctx, timer("t1", now.Add(-time.Minute), "o1")); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	// Idempotent re-schedule: same ID left untouched.
	if err := store.Schedule(ctx, timer("t1", now.Add(-time.Hour), "overwritten")); err != nil {
		t.Fatalf("re-schedule: %v", err)
	}

	mustSchedule(t, store, timer("late", now.Add(time.Hour), "late"))
	mustSchedule(t, store, timer("b", now.Add(-2*time.Second), "b"))
	mustSchedule(t, store, timer("a", now.Add(-2*time.Second), "a"))

	due, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("due: %v", err)
	}

	order := make([]string, 0, len(due))
	for _, tm := range due {
		order = append(order, tm.ID.Get())
	}

	// t1 (earliest), a, b (tie → ID order); 'late' gated.
	want := []string{"t1", "a", "b"}
	if len(order) != len(want) {
		t.Fatalf("due order = %v, want %v", order, want)
	}

	for i, id := range want {
		if order[i] != id {
			t.Fatalf("due order = %v, want %v (FireAt asc, ID tie-break)", order, want)
		}
	}

	if due[0].Payload.OrderID != "o1" {
		t.Fatalf("idempotent schedule was overwritten: %+v", due[0].Payload)
	}

	// Fire-once: before MarkFired, nothing is due again.
	again, err := store.Due(ctx, now)
	mustT(t, err)

	if len(again) != 0 {
		t.Fatalf("claimed timers must not re-fire before MarkFired: %d", len(again))
	}

	for _, tm := range due {
		if err := store.MarkFired(ctx, tm.ID); err != nil {
			t.Fatalf("mark fired: %v", err)
		}
	}

	after, err := store.Due(ctx, now.Add(2*time.Hour))
	mustT(t, err)

	if len(after) != 1 || after[0].ID.Get() != "late" {
		t.Fatalf("after fired: %+v", after)
	}
}

func TestTimerStore_MarkFiredEpochGuard(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()
	now := time.UnixMilli(1000)

	mustSchedule(t, store, timer("t", now.Add(-time.Minute), "gen1"))

	due, err := store.Due(ctx, now)
	mustT(t, err)

	if len(due) != 1 {
		t.Fatalf("due: %d", len(due))
	}

	// The documented race: while the dispatch of generation 1 is in flight,
	// the same ID is canceled and re-scheduled with a new deadline. The
	// stale MarkFired(gen1) must not delete generation 2.
	if err := store.Cancel(ctx, due[0].ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	mustSchedule(t, store, timer("t", now.Add(time.Hour), "gen2"))

	if err := store.MarkFired(ctx, due[0].ID); err != nil {
		t.Fatalf("stale mark fired: %v", err)
	}

	after, err := store.Due(ctx, now.Add(2*time.Hour))
	mustT(t, err)

	if len(after) != 1 || after[0].Payload.OrderID != "gen2" {
		t.Fatalf("stale MarkFired deleted the re-scheduled generation: %+v", after)
	}
}

func TestTimerStore_LeaseReclaim(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:schedengine_lease_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	first, err := engine.NewTimerStore[payload](eng,
		engine.WithOwner("dispatcher-1"), engine.WithLease(time.Minute))
	mustT(t, err)

	second, err := engine.NewTimerStore[payload](eng,
		engine.WithOwner("dispatcher-2"), engine.WithLease(time.Minute))
	mustT(t, err)

	ctx := context.Background()
	now := time.UnixMilli(1000)

	mustT(t, first.Schedule(ctx, timer("shared", now.Add(-time.Minute), "work")))

	due1, err := first.Due(ctx, now)
	mustT(t, err)

	if len(due1) != 1 {
		t.Fatalf("first dispatcher: %d", len(due1))
	}

	// Inside the lease the second dispatcher sees nothing.
	due2, err := second.Due(ctx, now)
	mustT(t, err)

	if len(due2) != 0 {
		t.Fatalf("lease fence violated: %+v", due2)
	}

	// After the lease lapses, the crashed dispatcher's timer is reclaimable.
	due2, err = second.Due(ctx, now.Add(61*time.Second))
	mustT(t, err)

	if len(due2) != 1 || due2[0].ID.Get() != "shared" {
		t.Fatalf("lease-expiry reclaim failed: %+v", due2)
	}
}

func TestTimerStore_ConcurrentDispatchersDisjoint(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:schedengine_race_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	now := time.UnixMilli(1000)

	seeder, err := engine.NewTimerStore[payload](eng)
	mustT(t, err)

	for i := range 30 {
		mustT(t, seeder.Schedule(ctx, timer(fmt.Sprintf("k%02d", i), now.Add(-time.Minute), "w")))
	}

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		got   = map[string]int{}
		start = make(chan struct{})
	)

	for worker := range 4 {
		store, err := engine.NewTimerStore[payload](
			eng,
			engine.WithOwner(fmt.Sprintf("w%d", worker)),
		)
		mustT(t, err)

		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			due, err := store.Due(ctx, now)
			if err != nil {
				t.Errorf("due: %v", err)

				return
			}

			mu.Lock()
			defer mu.Unlock()

			for _, tm := range due {
				got[tm.ID.Get()]++
			}
		}()
	}

	close(start)
	wg.Wait()

	if len(got) != 30 {
		t.Fatalf("dispatchers received %d distinct timers, want 30", len(got))
	}

	for id, count := range got {
		if count != 1 {
			t.Fatalf("timer %s dispatched %d times — double-fire", id, count)
		}
	}
}

func TestTimerStore_RejectsIncapableEngine(t *testing.T) {
	t.Parallel()

	if _, err := engine.NewTimerStore[payload](bareEngine{}); err == nil {
		t.Fatal("NewTimerStore must reject engines without DueClaimer")
	}
}

type bareEngine struct{}

func (bareEngine) Profile() metaengine.EngineProfile { return metaengine.EngineProfile{Name: "bare"} }
func (bareEngine) Close() error                      { return nil }

func mustSchedule(t *testing.T, s *engine.TimerStore[payload], tm scheduling.Timer[payload]) {
	t.Helper()

	if err := s.Schedule(t.Context(), tm); err != nil {
		t.Fatalf("schedule %s: %v", tm.ID.Get(), err)
	}
}

func mustT(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
