package scheduling_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TestScheduler_RejectionIsNotRetried pins the family-aware retry fix
// (T08): a dispatch error classified as Rejection or Conflict is a
// permanent decision — one attempt, no retry storm — while unclassified
// errors keep the retry behavior (TestScheduler_RetriesFailedDispatch).
func TestScheduler_RejectionIsNotRetried(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
	}{
		{"rejection", errorfamily.NewRejection("test.rejected", "no")},
		{"conflict", errorfamily.NewConflict("test.conflict", "version clash")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := scheduling.NewMemoryTimerStore[string]()

			timer := scheduling.Timer[string]{
				ID:      scheduling.MustParseTimerID("perm-" + tc.name),
				FireAt:  time.Now().Add(-1 * time.Second),
				Payload: "permanent-failure",
			}

			if err := store.Schedule(t.Context(), timer); err != nil {
				t.Fatalf("schedule: %v", err)
			}

			var attempts atomic.Int64

			sched := scheduling.New(
				store,
				func(_ context.Context, _ scheduling.Timer[string]) error {
					attempts.Add(1)

					return tc.err
				},
				scheduling.WithPollInterval(50*time.Millisecond),
				scheduling.WithMaxRetries(5),
				scheduling.WithRetryDelay(time.Millisecond),
			)

			runCtx, cancel := context.WithCancel(context.Background())

			go sched.Start(runCtx)
			defer cancel()

			// ~2-3 poll cycles elapse in 120ms (poll 50ms); each cycle
			// dispatches a stays-due timer exactly ONCE. The family-blind
			// loop would burn all MaxRetries=5 attempts inside the FIRST
			// cycle (retry delay 1ms), so anything above 3 proves retrying.
			time.Sleep(120 * time.Millisecond)

			if got := attempts.Load(); got == 0 || got > 3 {
				t.Fatalf(
					"permanent failure dispatched %d times in ~2 cycles, want 1-3 (family-blind would reach 5 in one)",
					got,
				)
			}
		})
	}
}

// TestScheduler_WrappedRejectionIsNotRetried: classification unwraps.
func TestScheduler_WrappedRejectionIsNotRetried(t *testing.T) {
	t.Parallel()

	store := scheduling.NewMemoryTimerStore[string]()

	if err := store.Schedule(t.Context(), scheduling.Timer[string]{
		ID:      scheduling.MustParseTimerID("wrapped"),
		FireAt:  time.Now().Add(-1 * time.Second),
		Payload: "wrapped",
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	var attempts atomic.Int64

	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			attempts.Add(1)

			return errorfamily.WrapRejection(errFail, "test.dispatch", "handler refused")
		},
		scheduling.WithPollInterval(50*time.Millisecond),
		scheduling.WithMaxRetries(5),
	)

	runCtx, cancel := context.WithCancel(context.Background())

	go sched.Start(runCtx)
	defer cancel()

	time.Sleep(120 * time.Millisecond)

	if got := attempts.Load(); got == 0 || got > 3 {
		t.Fatalf("wrapped rejection dispatched %d times in ~2 cycles, want 1-3", got)
	}
}
