package scheduling_test

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v3"
	"github.com/larsartmann/go-cqrs-lite/testutil/v3"
)

func TestMemoryTimerStore_ScheduleAndDue(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	now := time.Now()
	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "a", FireAt: now.Add(-1 * time.Minute), Payload: "early"},
	)
	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "b", FireAt: now.Add(1 * time.Hour), Payload: "late"},
	)

	due, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due timer, got %d", len(due))
	}
	if due[0].ID != "a" {
		t.Fatalf("expected 'a', got %q", due[0].ID)
	}
}

func TestMemoryTimerStore_ScheduleIsIdempotent(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "dup", FireAt: time.Now().Add(-1 * time.Minute)},
	)
	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "dup", FireAt: time.Now().Add(-1 * time.Minute)},
	)

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 1 {
		t.Fatalf("expected 1 timer (idempotent), got %d", len(due))
	}
}

func TestMemoryTimerStore_Cancel(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "cancel-me", FireAt: time.Now().Add(-1 * time.Minute)},
	)
	store.Cancel(ctx, "cancel-me")

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 0 {
		t.Fatalf("expected 0 after cancel, got %d", len(due))
	}
}

func TestMemoryTimerStore_MarkFired(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(
		ctx,
		scheduling.Timer[string]{ID: "fire", FireAt: time.Now().Add(-1 * time.Minute)},
	)
	store.MarkFired(ctx, "fire")

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 0 {
		t.Fatalf("expected 0 after mark fired, got %d", len(due))
	}
}

func TestScheduler_DispatchesDueTimers(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(ctx, scheduling.Timer[string]{
		ID:      "task-1",
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "run-me",
	})

	var dispatched atomic.Int64
	sched := scheduling.New(
		store,
		func(_ context.Context, timer scheduling.Timer[string]) error {
			if timer.ID != "task-1" {
				t.Errorf("expected task-1, got %s", timer.ID)
			}
			dispatched.Add(1)

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return dispatched.Load() == 1 })
	cancel()

	if dispatched.Load() != 1 {
		t.Fatalf("expected 1 dispatch, got %d", dispatched.Load())
	}
}

func TestScheduler_RetriesFailedDispatch(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(ctx, scheduling.Timer[string]{
		ID:      "retry-me",
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "fail-then-succeed",
	})

	var attempts atomic.Int64
	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			if attempts.Add(1) < 2 {
				return errFail
			}

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(3),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return attempts.Load() >= 2 })
	cancel()
}

var errFail = errStr("fail")

type errStr string

func (e errStr) Error() string { return string(e) }

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestScheduler_RetryDelayIsRespected(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(ctx, scheduling.Timer[string]{
		ID:      "delay-test",
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "will-succeed-on-retry",
	})

	const retryDelay = 50 * time.Millisecond

	var (
		attempts atomic.Int64
		firstAt  atomic.Int64 // unix-nano of first attempt
		secondAt atomic.Int64 // unix-nano of second attempt
	)

	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			n := attempts.Add(1)
			now := time.Now().UnixNano()
			if n == 1 {
				firstAt.Store(now)

				return errFail
			}
			if n == 2 {
				secondAt.Store(now)
			}

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(3),
		scheduling.WithRetryDelay(retryDelay),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return attempts.Load() >= 2 })
	cancel()

	elapsed := time.Duration(secondAt.Load() - firstAt.Load())
	// Equal jitter guarantees a minimum delay of cap/2 = 25ms for retryDelay=50ms.
	minExpected := retryDelay / 2
	if elapsed < minExpected {
		t.Fatalf("retry delay not respected: elapsed %v < expected min %v", elapsed, minExpected)
	}
}

func TestScheduler_WithLoggerIsUsed(t *testing.T) {
	t.Parallel()
	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(ctx, scheduling.Timer[string]{
		ID:      "always-fails",
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "boom",
	})

	capture := testutil.NewCapturingSlogHandler(slog.LevelError)
	logger := slog.New(capture)

	sched := scheduling.New(
		store, func(_ context.Context, _ scheduling.Timer[string]) error {
			return errFail
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(1),
		scheduling.WithLogger(logger),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return capture.Count() > 0 })
	cancel()

	if capture.Count() == 0 {
		t.Fatal("WithLogger: expected the injected logger to receive an error record, got none")
	}
}

func TestScheduler_FailedTimerSurvivesForRetry(t *testing.T) {
	t.Parallel()

	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	store.Schedule(ctx, scheduling.Timer[string]{
		ID:      "permanently-failing",
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "never-succeeds",
	})

	var attempts atomic.Int64
	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			attempts.Add(1)

			return errFail
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(2),
		scheduling.WithRetryDelay(time.Millisecond),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return attempts.Load() >= 3 })
	cancel()

	if attempts.Load() < 3 {
		t.Fatalf(
			"timer was deleted after failure; expected ≥3 attempts across polls, got %d",
			attempts.Load(),
		)
	}
}
