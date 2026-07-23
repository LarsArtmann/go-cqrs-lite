package integration_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ---------------------------------------------------------------------------
// 3. Million Aggregates — decider Execute + Load at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_DeciderExecute_ManyAggregates(b *testing.B) {
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
}

func BenchmarkScale_DeciderExecute_1000Aggregates_100UpdatesEach(b *testing.B) {
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
}

func BenchmarkScale_DeciderLoad_10KAggregates(b *testing.B) {
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
		for _, aggID := range aggIDs {
			_, _, err := repo.Load(ctx, aggID, "Item")
			if err != nil {
				b.Fatalf("Load: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
}
