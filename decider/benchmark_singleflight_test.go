package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func setupBenchStore(b *testing.B) (id.AggregateID, *eventtest.FakeStore) {
	b.Helper()

	store := eventtest.NewFakeStore()
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent(event.Type("CounterCreated"), aggID, "Counter", 1, []byte("{}"))
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}

	err = store.AppendBatch(
		context.Background(),
		event.NewAggregateRef("Counter", aggID),
		[]event.Event{evt},
	)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	return aggID, store
}

// BenchmarkLoad_Coalesced measures Load throughput with singleflight enabled
// (default). Concurrent loads of the same aggregate share one store.Load call.
func BenchmarkLoad_Coalesced(b *testing.B) {
	aggID, store := setupBenchStore(b)
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = repo.Load(ctx, aggID, "Counter")
		}
	})
}

// BenchmarkLoad_NoCoalescing measures Load throughput with singleflight
// disabled. Each goroutine calls store.Load independently.
func BenchmarkLoad_NoCoalescing(b *testing.B) {
	aggID, store := setupBenchStore(b)
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store,
		bus,
		d,
		decider.WithLoadCoalescing[counterState](false),
	)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = repo.Load(ctx, aggID, "Counter")
		}
	})
}
