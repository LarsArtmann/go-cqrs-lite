package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func benchEvent(
	tb testing.TB,
	eventType string,
	aggID id.AggregateID,
	version event.Version,
) *event.Core {
	tb.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", version, []byte("{}"))
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func BenchmarkDecider_Execute(b *testing.B) {
	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()

	for b.Loop() {
		aggID := id.NewAggregateID()

		err = repo.Execute(
			ctx, aggID, "Counter",
			func(_ counterState, v event.Version) ([]event.Event, error) {
				return []event.Event{benchEvent(b, "CounterCreated", aggID, v.Increment())}, nil
			},
		)
		if err != nil {
			b.Fatalf("Execute: %v", err)
		}
	}
}

func BenchmarkDecider_Execute_Update(b *testing.B) {
	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()

	for i := range 100 {
		err = repo.Execute(
			ctx, aggID, "Counter",
			func(_ counterState, v event.Version) ([]event.Event, error) {
				return []event.Event{benchEvent(b, "CounterIncremented", aggID, v.Increment())}, nil
			},
		)
		if err != nil {
			b.Fatalf("setup Execute %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		err = repo.Execute(
			ctx, aggID, "Counter",
			func(_ counterState, v event.Version) ([]event.Event, error) {
				return []event.Event{benchEvent(b, "CounterIncremented", aggID, v.Increment())}, nil
			},
		)
		if err != nil {
			b.Fatalf("Execute update: %v", err)
		}
	}
}

func BenchmarkDecider_Load(b *testing.B) {
	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()

	for i := range 100 {
		err = repo.Execute(
			ctx, aggID, "Counter",
			func(_ counterState, v event.Version) ([]event.Event, error) {
				return []event.Event{benchEvent(b, "CounterIncremented", aggID, v.Increment())}, nil
			},
		)
		if err != nil {
			b.Fatalf("setup Execute %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		_, _, err = repo.Load(ctx, aggID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkDecider_Fold(b *testing.B) {
	events := make([]event.Event, 100)
	aggID := id.NewAggregateID()

	for i := range 100 {
		events[i] = benchEvent(b, "CounterIncremented", aggID, event.Version(i+1))
	}

	state := counterState{Value: 0}

	b.ResetTimer()

	for b.Loop() {
		s := state

		for _, evt := range events {
			s, _ = foldCounter(s, evt)
		}
	}
}
