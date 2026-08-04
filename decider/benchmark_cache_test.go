package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func BenchmarkDecider_Load_NoCache(b *testing.B) {
	b.ReportAllocs()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	repo, err := decider.NewRepository(store, bus, counterDecider())
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()
	seedCounterBench(b, repo, streamID, 500)

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, streamID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkDecider_Load_WithCache(b *testing.B) {
	b.ReportAllocs()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		decider.WithStateCache[counterState](decider.NewStateCache[counterState](128)),
	)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()
	seedCounterBench(b, repo, streamID, 500)

	// Warm the cache with one Load
	_, _, err = repo.Load(ctx, streamID, "Counter")
	if err != nil {
		b.Fatalf("Load (warm): %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		state, _, err := repo.Load(ctx, streamID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
		if state.Value != 500 {
			b.Fatalf("Load: state.Value=%d, want 500", state.Value)
		}
	}
}

func BenchmarkDecider_Load_WithCache_HeavyHistory(b *testing.B) {
	b.ReportAllocs()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		decider.WithStateCache[counterState](decider.NewStateCache[counterState](128)),
	)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()

	// Seed 5000 events — heavy history where cache benefit is maximal
	for i := 0; i < 5000; i++ {
		benchExecute(b, repo, ctx, streamID, "CounterIncremented")
	}

	// Warm the cache
	_, _, err = repo.Load(ctx, streamID, "Counter")
	if err != nil {
		b.Fatalf("Load (warm): %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, streamID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkStateCache_Get(b *testing.B) {
	b.ReportAllocs()

	cache := decider.NewStateCache[counterState](128)
	ref := id.NewStreamRef("Counter", id.NewStreamID())
	cache.Put(ref, counterState{Value: 42}, event.Version(10))

	b.ResetTimer()

	for b.Loop() {
		_, _, found := cache.Get(ref)
		if !found {
			b.Fatal("cache.Get: expected found=true for existing key")
		}
	}
}

func BenchmarkStateCache_Put(b *testing.B) {
	b.ReportAllocs()

	cache := decider.NewStateCache[counterState](128)
	ref := id.NewStreamRef("Counter", id.NewStreamID())

	b.ResetTimer()

	for b.Loop() {
		cache.Put(ref, counterState{Value: 42}, event.Version(10))
	}

	val, _, found := cache.Get(ref)
	if !found || val.Value != 42 {
		b.Fatalf("post-loop Get: found=%v value=%d, want 42", found, val.Value)
	}
}
