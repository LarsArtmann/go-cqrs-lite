//go:build scale

package integration_test

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// ---------------------------------------------------------------------------
// 2. Projection Replay — 100K pre-stored events replayed through typed handlers
// ---------------------------------------------------------------------------

func BenchmarkRealistic_ProjectionReplay(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	aggCount := 1000
	eventsPerAgg := 100

	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	_ = seedOrders(b, store, aggCount, eventsPerAgg)
	totalEvents := countEvents(b, store)

	var processed atomic.Int64

	builder := projection.NewBuilder("replay-view")
	projection.On(
		builder,
		"OrderCreated",
		jsonCodec,
		func(_ context.Context, _ OrderCreated) error {
			processed.Add(1)
			return nil
		},
	)
	projection.On(builder, "ItemAdded", jsonCodec, func(_ context.Context, _ ItemAdded) error {
		processed.Add(1)
		return nil
	})
	projection.On(
		builder,
		"OrderShipped",
		jsonCodec,
		func(_ context.Context, _ OrderShipped) error {
			processed.Add(1)
			return nil
		},
	)

	b.ResetTimer()

	for b.Loop() {
		processed.Store(0)

		cp := memory.NewMemoryCheckpointStore()
		runner, err := projection.NewRunner(store, bus, cp)
		if err != nil {
			b.Fatalf("NewRunner: %v", err)
		}

		if err := runner.Register(builder.Build()); err != nil {
			b.Fatalf("Register: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			_ = runner.Run(ctx)
			close(done)
		}()

		for processed.Load() < int64(totalEvents) {
			runtime.Gosched()
		}

		cancel()
		<-done
		_ = runner.Close()
		_ = cp.Close()
	}

	b.ReportMetric(float64(totalEvents), "events")
	b.ReportMetric(float64(b.N*totalEvents)/b.Elapsed().Seconds(), "events-replayed/sec")
}
