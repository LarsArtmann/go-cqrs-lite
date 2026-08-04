package sqlstore_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

var propDBCounter atomic.Int64

// newPropStore creates an in-memory SQLite-backed timer store for property
// tests. Each call gets a unique database name to avoid cross-test interference.
func newPropStore[P any](tb testing.TB) *sqlstore.SQLTimerStore[P] {
	tb.Helper()

	n := propDBCounter.Add(1)
	dsn := fmt.Sprintf("file:proptimer%d?mode=memory&cache=shared&_pragma=busy_timeout(5000)", n)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}

	tb.Cleanup(func() { _ = db.Close() })

	store, err := sqlstore.NewSQLiteStore[P](context.Background(), db)
	if err != nil {
		tb.Fatalf("NewSQLiteStore: %v", err)
	}

	return store
}

// TestProperty_ScheduleIsIdempotent: Scheduling the same timer ID multiple
// times never errors and the timer remains with the original payload.
func TestProperty_ScheduleIsIdempotent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := newPropStore[testPayload](t)
		ctx := context.Background()

		id := "timer-" + rapid.String().Draw(rt, "id")
		fireAt := time.Now().
			Add(time.Duration(rapid.IntRange(1, 3600).Draw(rt, "seconds")) * time.Second)
		originalPayload := testPayload{
			Action: "original",
			Amount: rapid.IntRange(1, 1000).Draw(rt, "amount"),
		}

		for i := 0; i < rapid.IntRange(2, 20).Draw(rt, "repeats"); i++ {
			timer := scheduling.Timer[testPayload]{
				ID:      id,
				FireAt:  fireAt,
				Payload: originalPayload,
			}
			if err := store.Schedule(ctx, timer); err != nil {
				rt.Fatalf("Schedule attempt %d: %v", i, err)
			}
		}

		due, err := store.Due(ctx, fireAt.Add(1*time.Second))
		if err != nil {
			rt.Fatalf("Due: %v", err)
		}

		if len(due) != 1 {
			rt.Fatalf("expected 1 timer after idempotent schedule, got %d", len(due))
		}

		if due[0].Payload.Amount != originalPayload.Amount {
			rt.Fatalf(
				"idempotent schedule must keep original payload; got %d, want %d",
				due[0].Payload.Amount,
				originalPayload.Amount,
			)
		}
	})
}

// TestProperty_ConcurrentScheduleSameID: N goroutines concurrently schedule
// the same timer ID. Exactly one timer should exist afterward (no duplicates,
// no corruption).
func TestProperty_ConcurrentScheduleSameID(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := newPropStore[testPayload](t)
		ctx := context.Background()

		id := "concurrent-" + rapid.String().Draw(rt, "id")
		n := rapid.IntRange(2, 12).Draw(rt, "goroutines")
		fireAt := time.Now().Add(1 * time.Hour)

		var wg sync.WaitGroup
		errs := make(chan error, n)

		for i := 0; i < n; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				errs <- store.Schedule(ctx, scheduling.Timer[testPayload]{
					ID:      id,
					FireAt:  fireAt,
					Payload: testPayload{Action: "concurrent", Amount: i},
				})
			}(i)
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				rt.Fatalf("concurrent Schedule returned error: %v", err)
			}
		}

		due, err := store.Due(ctx, fireAt.Add(1*time.Second))
		if err != nil {
			rt.Fatalf("Due: %v", err)
		}

		if len(due) != 1 {
			rt.Fatalf("expected exactly 1 timer after concurrent schedule, got %d", len(due))
		}
	})
}

// TestProperty_DueOrdering: After scheduling multiple timers with different
// FireAt values, Due always returns them in ascending FireAt order.
func TestProperty_DueOrdering(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := newPropStore[testPayload](t)
		ctx := context.Background()
		base := time.Now().UTC()

		n := rapid.IntRange(2, 30).Draw(rt, "timer_count")

		for i := 0; i < n; i++ {
			offset := time.Duration(rapid.IntRange(1, 3600).Draw(rt, "offset")) * time.Second
			if err := store.Schedule(ctx, scheduling.Timer[testPayload]{
				ID:      fmt.Sprintf("timer-%d", i),
				FireAt:  base.Add(offset),
				Payload: testPayload{Action: "test", Amount: i},
			}); err != nil {
				rt.Fatalf("Schedule %d: %v", i, err)
			}
		}

		due, err := store.Due(ctx, base.Add(2*time.Hour))
		if err != nil {
			rt.Fatalf("Due: %v", err)
		}

		if len(due) != n {
			rt.Fatalf("expected %d due timers, got %d", n, len(due))
		}

		for i := 1; i < len(due); i++ {
			if due[i].FireAt.Before(due[i-1].FireAt) {
				rt.Fatalf(
					"Due not sorted ascending: [%d].FireAt=%v < [%d].FireAt=%v",
					i, due[i].FireAt, i-1, due[i-1].FireAt,
				)
			}
		}
	})
}

// TestProperty_MarkFiredRemovesTimer: MarkFired always removes the timer so
// subsequent Due calls never return it.
func TestProperty_MarkFiredRemovesTimer(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := newPropStore[testPayload](t)
		ctx := context.Background()

		id := "fire-" + rapid.String().Draw(rt, "id")
		fireAt := time.Now().Add(1 * time.Hour)

		if err := store.Schedule(ctx, scheduling.Timer[testPayload]{
			ID:      id,
			FireAt:  fireAt,
			Payload: testPayload{Action: "go"},
		}); err != nil {
			rt.Fatalf("Schedule: %v", err)
		}

		if err := store.MarkFired(ctx, id); err != nil {
			rt.Fatalf("MarkFired: %v", err)
		}

		due, err := store.Due(ctx, fireAt.Add(1*time.Hour))
		if err != nil {
			rt.Fatalf("Due: %v", err)
		}

		for _, tm := range due {
			if tm.ID == id {
				rt.Fatalf("timer %s should be gone after MarkFired", id)
			}
		}
	})
}

// TestProperty_ConcurrentScheduleAndMarkFired: Schedule and MarkFired running
// concurrently on different timers should never panic, corrupt, or deadlock.
func TestProperty_ConcurrentScheduleAndMarkFired(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := newPropStore[testPayload](t)
		ctx := context.Background()
		base := time.Now()

		numTimers := rapid.IntRange(5, 20).Draw(rt, "timers")

		var wg sync.WaitGroup

		// Schedule timers
		for i := 0; i < numTimers; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				_ = store.Schedule(ctx, scheduling.Timer[testPayload]{
					ID:      fmt.Sprintf("race-%d", i),
					FireAt:  base.Add(time.Duration(i) * time.Second),
					Payload: testPayload{Action: "race", Amount: i},
				})
			}(i)
		}

		// Concurrently fire some timers while scheduling is ongoing
		for i := 0; i < numTimers/2; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				_ = store.MarkFired(ctx, fmt.Sprintf("race-%d", i))
				_, _ = store.Due(ctx, base.Add(1*time.Hour))
			}(i)
		}

		wg.Wait()

		// Final state should be queryable without error
		_, err := store.Due(ctx, base.Add(1*time.Hour))
		if err != nil {
			rt.Fatalf("Due after concurrent ops: %v", err)
		}
	})
}
