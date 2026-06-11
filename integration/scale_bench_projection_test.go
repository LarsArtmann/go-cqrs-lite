package integration_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// ---------------------------------------------------------------------------
// 4. Thousand Projections — registration + event processing at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_ProjectionRegistration_1000(b *testing.B) {
	b.ReportAllocs()

	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	b.ResetTimer()

	var counter atomic.Int64

	for b.Loop() {
		n := counter.Add(1)
		p := event.NewProjection(
			fmt.Sprintf("projection-%d", n),
			noopEventHandler(),
			[]event.Type{"ItemCreated"},
		)

		err := runner.Register(p)
		if err != nil {
			b.Fatalf("Register: %v", err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "registrations/sec")
}

func BenchmarkScale_ProjectionProcessing_100Projections_100KEvents(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	projectionCount := 100
	eventCount := 100_000

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	var processed atomic.Int64

	for i := range projectionCount {
		counter := &processed
		p := event.NewProjection(
			fmt.Sprintf("view-%d", i),
			func(_ context.Context, _ event.Event) error {
				counter.Add(1)

				return nil
			},
			[]event.Type{"ItemCreated", "ItemUpdated"},
		)

		err := runner.Register(p)
		if err != nil {
			b.Fatalf("Register: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)

	go func() {
		runErr <- runner.Run(ctx)
	}()

	seedCtx := context.Background()
	aggID := id.NewAggregateID()

	for v := range eventCount {
		evt := newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))

		err := bus.Publish(seedCtx, evt)
		if err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}

	b.ResetTimer()

	start := time.Now()

	for b.Loop() {
		processed.Store(0)
		aggID := id.NewAggregateID()

		for v := range eventCount {
			evt := newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))
			_ = bus.Publish(seedCtx, evt)
		}
	}

	elapsed := time.Since(start)
	b.ReportMetric(
		float64(b.N*eventCount*projectionCount)/elapsed.Seconds(),
		"projection-events/sec",
	)

	cancel()
	_ = runner.Close()
}
