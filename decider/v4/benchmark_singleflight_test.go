package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func setupBenchStore(b *testing.B) (id.StreamID, *eventtest.FakeStore) {
	b.Helper()

	store := eventtest.NewFakeStore()
	streamID := id.NewStreamID()

	evt, err := event.NewEvent(event.Type("CounterCreated"), streamID, "Counter", 1, []byte("{}"))
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}

	err = store.AppendBatch(
		context.Background(),
		id.NewStreamRef("Counter", streamID),
		[]event.Event{evt},
	)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	return streamID, store
}

// BenchmarkLoad_Coalesced measures Load throughput with singleflight enabled
// (default). Concurrent loads of the same stream share one store.Load call.
func BenchmarkLoad_Coalesced(b *testing.B) {
	streamID, store := setupBenchStore(b)
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			state, _, err := repo.Load(ctx, streamID, "Counter")
			if err != nil {
				b.Fatalf("Load: %v", err)
			}
			if state.Value != 1 {
				b.Fatalf("Load: state.Value=%d, want 1", state.Value)
			}
		}
	})
}

// BenchmarkLoad_NoCoalescing measures Load throughput with singleflight
// disabled. Each goroutine calls store.Load independently.
func BenchmarkLoad_NoCoalescing(b *testing.B) {
	streamID, store := setupBenchStore(b)
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
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
			state, _, err := repo.Load(ctx, streamID, "Counter")
			if err != nil {
				b.Fatalf("Load: %v", err)
			}
			if state.Value != 1 {
				b.Fatalf("Load: state.Value=%d, want 1", state.Value)
			}
		}
	})
}
