package integration_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ---------------------------------------------------------------------------
// 3. Million Streams — decider Execute + Load at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_DeciderExecute_ManyStreams(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggIDs := make([]id.StreamID, b.N)
	for i := range aggIDs {
		aggIDs[i] = id.NewStreamID()
	}

	b.ResetTimer()

	for i := range aggIDs {
		benchCreateItem(b, repo, ctx, aggIDs[i])
	}

	// Verify events were persisted (not silently dropped).
	state, _, err := repo.Load(ctx, aggIDs[0], "Item")
	if err != nil {
		b.Fatalf("post-loop Load: %v", err)
	}
	if state.Value < 1 {
		b.Fatalf("post-loop Load: state=%+v, expected Value >= 1 — Execute was a no-op", state)
	}
}

func BenchmarkScale_DeciderExecute_1000Streams_100UpdatesEach(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggCount := 1_000
	aggIDs := make([]id.StreamID, aggCount)

	for i := range aggIDs {
		aggIDs[i] = id.NewStreamID()

		benchCreateItem(b, repo, ctx, aggIDs[i])
	}

	b.ResetTimer()

	for b.Loop() {
		for i := range aggIDs {
			benchCreateItem(b, repo, ctx, aggIDs[i])
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "executes/sec")

	// Verify events persisted after all iterations.
	state, _, err := repo.Load(ctx, aggIDs[0], "Item")
	if err != nil {
		b.Fatalf("post-loop Load: %v", err)
	}
	if state.Value < 1 {
		b.Fatalf("post-loop Load: state=%+v, expected events — Execute was a no-op", state)
	}
}

func BenchmarkScale_DeciderLoad_10KStreams(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggCount := 10_000
	eventsPerAgg := 50
	aggIDs := make([]id.StreamID, aggCount)

	for i := range aggIDs {
		aggIDs[i] = id.NewStreamID()

		for range eventsPerAgg {
			err := repo.Execute(
				ctx, aggIDs[i], "Item",
				func(_ benchState, ver event.Version) ([]event.Event, error) {
					return []event.Event{
						newBenchEvent(b, "ItemUpdated", aggIDs[i], ver.Increment()),
					}, nil
				},
			)
			if err != nil {
				b.Fatalf("seed Execute: %v", err)
			}
		}
	}

	b.ResetTimer()

	for b.Loop() {
		for _, streamID := range aggIDs {
			state, _, err := repo.Load(ctx, streamID, "Item")
			if err != nil {
				b.Fatalf("Load: %v", err)
			}
			if state.Value < 1 {
				b.Fatalf("Load: state=%+v for stream %s, expected events — store was empty", state, streamID)
			}
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
}
