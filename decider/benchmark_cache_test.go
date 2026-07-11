package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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
	aggID := id.NewAggregateID()
	seedCounterBench(b, repo, aggID, 500)

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, aggID, "Counter")
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
	aggID := id.NewAggregateID()
	seedCounterBench(b, repo, aggID, 500)

	// Warm the cache with one Load
	_, _, err = repo.Load(ctx, aggID, "Counter")
	if err != nil {
		b.Fatalf("Load (warm): %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, aggID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
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
	aggID := id.NewAggregateID()

	// Seed 5000 events — heavy history where cache benefit is maximal
	for i := 0; i < 5000; i++ {
		benchExecute(b, repo, ctx, aggID, "CounterIncremented")
	}

	// Warm the cache
	_, _, err = repo.Load(ctx, aggID, "Counter")
	if err != nil {
		b.Fatalf("Load (warm): %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, aggID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkStateCache_Get(b *testing.B) {
	b.ReportAllocs()

	cache := decider.NewStateCache[counterState](128)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())
	cache.Put(ref, counterState{Value: 42}, event.Version(10))

	b.ResetTimer()

	for b.Loop() {
		_, _, _ = cache.Get(ref)
	}
}

func BenchmarkStateCache_Put(b *testing.B) {
	b.ReportAllocs()

	cache := decider.NewStateCache[counterState](128)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	b.ResetTimer()

	for b.Loop() {
		cache.Put(ref, counterState{Value: 42}, event.Version(10))
	}
}
