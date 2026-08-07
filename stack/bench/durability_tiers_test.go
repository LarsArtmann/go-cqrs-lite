package bench

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// durability_tiers_test.go — measures the fsync cost tradeoff across SQLite
// durability tiers. Strict = fsync per commit, Normal = WAL default,
// Relaxed = no fsync. Consumers need to see the latency cost of durability.

func BenchmarkDurabilityTiers_SQLite(b *testing.B) {
	for _, tier := range []stack.DurabilityTier{
		stack.DurabilityStrict,
		stack.DurabilityNormal,
		stack.DurabilityRelaxed,
	} {
		b.Run(string(tier), func(b *testing.B) {
			dir := b.TempDir()
			bundle, err := sqlite.New(
				dir+"/durability.db",
				sqlite.WithDurability(tier),
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
				ref := id.NewStreamRef("Bench", streamID)

				evt, err := event.NewEvent(
					"bench.event", streamID, "Bench", event.Version(1),
					[]byte(`{"data":"test"}`),
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkDurabilityTiers_BatchWrite measures batch write throughput across
// durability tiers. Batch writes should amortize the fsync cost.
func BenchmarkDurabilityTiers_BatchWrite(b *testing.B) {
	for _, tier := range []stack.DurabilityTier{
		stack.DurabilityStrict,
		stack.DurabilityNormal,
		stack.DurabilityRelaxed,
	} {
		b.Run(string(tier), func(b *testing.B) {
			dir := b.TempDir()
			bundle, err := sqlite.New(
				dir+"/durability_batch.db",
				sqlite.WithDurability(tier),
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
			batchSize := 10

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("Bench", streamID)

				events := make([]event.Event, batchSize)
				for i := range batchSize {
					evt, err := event.NewEvent(
						"bench.event", streamID, "Bench", event.Version(i+1),
						[]byte(`{"data":"test"}`),
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

// BenchmarkDurabilityTiers_Bbolt measures the fsync cost tradeoff across bbolt
// durability tiers. Strict/Normal = sync-on-commit, Relaxed = NoSync.
func BenchmarkDurabilityTiers_Bbolt(b *testing.B) {
	for _, tier := range []stack.DurabilityTier{
		stack.DurabilityStrict,
		stack.DurabilityNormal,
		stack.DurabilityRelaxed,
	} {
		b.Run(string(tier), func(b *testing.B) {
			dir := b.TempDir()
			bundle, err := bbolt.New(
				dir+"/durability.db",
				bbolt.WithDurability(tier),
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
				ref := id.NewStreamRef("Bench", streamID)

				evt, err := event.NewEvent(
					"bench.event", streamID, "Bench", event.Version(1),
					[]byte(`{"data":"test"}`),
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// suppress unused import.
var _ = time.Second
