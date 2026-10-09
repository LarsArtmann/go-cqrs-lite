package scheduling

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestScheduler_WithClock_FreezesDueWindow pins the injected time source
// (ADR-0153 D4): with a manual now, a timer due in one hour must NOT fire
// while the clock is frozen, and must fire once the clock advances past it.
func TestScheduler_WithClock_FreezesDueWindow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := base

	store := NewMemoryTimerStore[string]()
	if err := store.Schedule(ctx, Timer[string]{
		ID:      MustParseTimerID("clock-freeze"),
		FireAt:  now.Add(time.Hour),
		Payload: "go",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	var dispatched atomic.Int64

	sched := New(store,
		func(_ context.Context, _ Timer[string]) error {
			dispatched.Add(1)

			return nil
		},
		WithPollInterval(time.Millisecond),
		WithClock(func() time.Time { return now }),
	)

	go func() { _ = sched.Start(ctx) }()

	// Frozen clock: the scheduler polls but nothing is due.
	time.Sleep(50 * time.Millisecond)
	if got := dispatched.Load(); got != 0 {
		t.Fatalf("timer fired under frozen clock: %d dispatches", got)
	}

	// Advance past FireAt: the next poll must dispatch.
	now = now.Add(2 * time.Hour)

	deadline := time.Now().Add(3 * time.Second)
	for dispatched.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	if got := dispatched.Load(); got == 0 {
		t.Fatal("timer did not fire after the clock advanced past FireAt")
	}
}
