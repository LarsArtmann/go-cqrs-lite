package integration_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// ---------------------------------------------------------------------------
// 7. Full Pipeline — end-to-end: command → decider → event → projection → query
// ---------------------------------------------------------------------------

func BenchmarkScale_FullPipeline_1KAggregates(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	repo, err := decider.NewRepository(store, bus, benchDecider())
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	var projectionEvents atomic.Int64

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	counter := &projectionEvents
	proj := event.NewProjection(
		"full-pipeline",
		func(_ context.Context, _ event.Event) error {
			counter.Add(1)

			return nil
		},
		[]event.Type{"ItemCreated", "ItemUpdated"},
	)

	err = runner.Register(proj)
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() { _ = runner.Run(ctx) }()

	queryDisp := query.NewDispatcher()
	b.Cleanup(func() { _ = queryDisp.Close() })

	err = queryDisp.Register("list.items", func(_ context.Context, _ query.Query) (any, error) {
		return map[string]int{"total": int(projectionEvents.Load())}, nil
	})
	if err != nil {
		b.Fatalf("register query: %v", err)
	}

	aggCount := 1_000

	b.ResetTimer()

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
	}

	for b.Loop() {
		projectionEvents.Store(0)

		for _, aggID := range aggIDs {
			benchCreateItem(b, repo, context.Background(), aggID)
		}

		q := mustNewQuery("list.items")
		_, err := queryDisp.Dispatch(context.Background(), q)
		if err != nil {
			b.Fatalf("Dispatch: %v", err)
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "aggregates/sec")

	cancel()
	_ = runner.Close()
}
