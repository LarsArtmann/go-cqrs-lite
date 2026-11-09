package scheduling_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// bench_test.go — measures timer schedule/poll/dispatch throughput.

// BenchmarkScheduling_ScheduleAndPoll measures the full timer lifecycle:
// schedule N timers, poll for due timers, dispatch, mark fired.
func BenchmarkScheduling_ScheduleAndPoll(b *testing.B) {
	for _, n := range []int{100, 1_000} {
		b.Run(fmt.Sprintf("timers=%d", n), func(b *testing.B) {
			ctx := context.Background()
			now := time.Now()

			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				store := scheduling.NewMemoryTimerStore[string]()
				timers := make([]scheduling.Timer[string], n)
				for i := range n {
					timers[i] = scheduling.Timer[string]{
						ID:      scheduling.MustParseTimerID(fmt.Sprintf("timer-%d", i)),
						FireAt:  now.Add(time.Duration(i) * time.Microsecond),
						Payload: "dispatch-me",
					}
				}
				b.StartTimer()

				// Phase 1: Schedule all timers.
				for _, t := range timers {
					if err := store.Schedule(ctx, t); err != nil {
						b.Fatal(err)
					}
				}

				// Phase 2: Poll for due timers.
				due, err := store.Due(ctx, now.Add(time.Hour))
				if err != nil {
					b.Fatal(err)
				}
				if len(due) != n {
					b.Fatalf("expected %d due, got %d", n, len(due))
				}

				// Phase 3: Mark all as fired.
				for _, t := range due {
					if err := store.MarkFired(ctx, t.ID); err != nil {
						b.Fatal(err)
					}
				}
			}

			b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "timers/sec")
		})
	}
}

// BenchmarkScheduling_Cancel measures timer cancellation throughput.
func BenchmarkScheduling_Cancel(b *testing.B) {
	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()

	for b.Loop() {
		b.StopTimer()
		store := scheduling.NewMemoryTimerStore[string]()
		for i := range b.N {
			_ = store.Schedule(ctx, scheduling.Timer[string]{
				ID:     scheduling.MustParseTimerID(fmt.Sprintf("cancel-%d", i)),
				FireAt: now.Add(time.Hour),
			})
		}
		b.StartTimer()

		for i := range b.N {
			if err := store.Cancel(
				ctx,
				scheduling.MustParseTimerID(fmt.Sprintf("cancel-%d", i)),
			); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "cancels/sec")
}
