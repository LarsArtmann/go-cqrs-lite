//go:build scale

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// ---------------------------------------------------------------------------
// 6. Snapshot vs Replay — load 100 aggregates × 500 events each
// ---------------------------------------------------------------------------

func BenchmarkRealistic_SnapshotVsReplay(b *testing.B) {
	eventsPerAgg := 500
	aggCount := 100

	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	snapRepo := benchNewOrderRepo(b, store, bus, 100)

	ctx := context.Background()

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggCount {
		aggIDs[i] = id.NewAggregateID()

		for range eventsPerAgg {
			if err := snapRepo.Execute(ctx, aggIDs[i], "Order",
				func(_ OrderState, ver event.Version) ([]event.Event, error) {
					return []event.Event{
						newRealisticEvent(
							b,
							"OrderCreated",
							aggIDs[i],
							ver.Increment(),
							OrderCreated{
								OrderID:   aggIDs[i].String(),
								Customer:  "test",
								Total:     10.0,
								Items:     1,
								Timestamp: time.Now().Format(time.RFC3339),
							},
						),
					}, nil
				}); err != nil {
				b.Fatalf("seed: %v", err)
			}
		}
	}

	b.Run("Replay", func(b *testing.B) {
		b.ReportAllocs()

		plainDecider := decider.Decider[OrderState]{Initial: OrderState{}, Apply: applyOrder}
		replayRepo, _ := decider.NewRepository[OrderState](store, bus, plainDecider)

		b.ResetTimer()

		for b.Loop() {
			for _, aggID := range aggIDs {
				if _, _, err := replayRepo.Load(ctx, aggID, "Order"); err != nil {
					b.Fatalf("Load: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventsPerAgg), "events_per_agg")
		b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
	})

	b.Run("Snapshot", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for b.Loop() {
			for _, aggID := range aggIDs {
				if _, _, err := snapRepo.Load(ctx, aggID, "Order"); err != nil {
					b.Fatalf("Load: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventsPerAgg), "events_per_agg")
		b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
	})
}
