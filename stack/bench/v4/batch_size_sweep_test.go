package bench

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// batch_size_sweep_test.go — measures how batch size affects write throughput.
// Real systems batch events for performance. Where is the sweet spot?

// BenchmarkBatchSizeSweep_SQLite measures single-stream batch write throughput
// across different batch sizes on SQLite.
func BenchmarkBatchSizeSweep_SQLite(b *testing.B) {
	for _, batchSize := range []int{1, 5, 10, 50, 100} {
		b.Run(fmt.Sprintf("batch=%d", batchSize), func(b *testing.B) {
			dir := b.TempDir()
			bundle, err := sqlite.New(
				dir+"/batch.db",
				sqlite.WithPragmas(sqlopt.WithOptimizations()),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = bundle.Close() }()

			store, ok := bundle.EventStore()
			if !ok {
				b.Fatal("no event store")
			}

			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("Batch", streamID)

				events := make([]event.Event, batchSize)
				for i := range batchSize {
					evt, err := event.NewEvent(
						"batch.event", streamID, "Batch", event.Version(i+1),
						[]byte(`{"n":1}`),
					)
					if err != nil {
						b.Fatal(err)
					}
					events[i] = evt
				}
				if err := store.AppendBatch(ctx, ref, events); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkBatchSizeSweep_Pebble measures batch write throughput on Pebble.
func BenchmarkBatchSizeSweep_Pebble(b *testing.B) {
	for _, batchSize := range []int{1, 5, 10, 50, 100} {
		b.Run(fmt.Sprintf("batch=%d", batchSize), func(b *testing.B) {
			pb := pebbleBackend()
			bundle, cleanup := pb.create(b)
			defer cleanup()

			store, ok := bundle.EventStore()
			if !ok {
				b.Fatal("no event store")
			}

			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("Batch", streamID)

				events := make([]event.Event, batchSize)
				for i := range batchSize {
					evt, err := event.NewEvent(
						"batch.event", streamID, "Batch", event.Version(i+1),
						[]byte(`{"n":1}`),
					)
					if err != nil {
						b.Fatal(err)
					}
					events[i] = evt
				}
				if err := store.AppendBatch(ctx, ref, events); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}
