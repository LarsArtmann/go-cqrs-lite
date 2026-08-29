package scheduling_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// corruptingStore wraps a TimerStore and reports a joined corruption error
// from Due while still returning the decodable timers, mirroring
// SQLTimerStore.Due's skip-and-join behavior for rotten rows.
type corruptingStore struct {
	inner scheduling.TimerStore[string]
}

func (s *corruptingStore) Schedule(
	ctx context.Context,
	t scheduling.Timer[string],
) error {
	return s.inner.Schedule(ctx, t)
}

func (s *corruptingStore) Due(
	ctx context.Context,
	now time.Time,
) ([]scheduling.Timer[string], error) {
	timers, _ := s.inner.Due(ctx, now)

	return timers, errors.New("scheduling.sqlstore.decode: row 42: payload corrupt")
}

func (s *corruptingStore) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	return s.inner.MarkFired(ctx, id)
}

func (s *corruptingStore) Cancel(ctx context.Context, id scheduling.TimerID) error {
	return s.inner.Cancel(ctx, id)
}

// TestScheduler_MaxRetriesZeroStillDispatchesOnce pins the WithMaxRetries
// clamp: a 0 (read as "no retries") must not become "no attempts" — the old
// behavior marked the timer fired with zero dispatches, permanently losing
// the deadline.
func TestScheduler_MaxRetriesZeroStillDispatchesOnce(t *testing.T) {
	t.Parallel()

	store := scheduling.NewMemoryTimerStore[string]()
	ctx := context.Background()

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID:      scheduling.MustParseTimerID("clamp-me"),
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "fires",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	var attempts atomic.Int64
	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			attempts.Add(1)

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(0),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return attempts.Load() >= 1 })
	cancel()

	if attempts.Load() < 1 {
		t.Fatalf(
			"WithMaxRetries(0) dispatched %d times, want >= 1 (deadline must not be lost)",
			attempts.Load(),
		)
	}
}

// TestScheduler_DispatchesDespiteCorruptRows pins the head-of-line fix: a Due
// error returned WITH decodable timers must not block dispatch — one rotten
// row must not stop every other due timer.
func TestScheduler_DispatchesDespiteCorruptRows(t *testing.T) {
	t.Parallel()

	store := &corruptingStore{inner: scheduling.NewMemoryTimerStore[string]()}
	ctx := context.Background()

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID:      scheduling.MustParseTimerID("healthy"),
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "fires",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	var attempts atomic.Int64
	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			attempts.Add(1)

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithRetryDelay(time.Millisecond),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitFor(t, 2*time.Second, func() bool { return attempts.Load() >= 1 })
	cancel()

	if attempts.Load() < 1 {
		t.Fatal(
			"healthy timer was not dispatched despite being decodable; corrupt row blocked dispatch",
		)
	}
}
