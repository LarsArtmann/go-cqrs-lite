package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func benchEvent(
	tb testing.TB,
	eventType string,
	aggID id.AggregateID,
	version event.Version,
) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", version, []byte("{}"))
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func seedCounterBench(
	b *testing.B,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	n int,
) {
	b.Helper()

	ctx := context.Background()

	for range n {
		benchExecute(b, repo, ctx, aggID, "CounterIncremented")
	}
}

func benchExecute(
	b *testing.B,
	repo *decider.Repository[counterState],
	ctx context.Context,
	aggID id.AggregateID,
	eventType string,
) {
	b.Helper()

	err := repo.Execute(
		ctx, aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{benchEvent(b, eventType, aggID, v.Increment())}, nil
		},
	)
	if err != nil {
		b.Fatalf("Execute(%s): %v", eventType, err)
	}
}

func BenchmarkDecider_Execute(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)

	for b.Loop() {
		aggID := id.NewAggregateID()
		benchExecute(b, repo, ctx, aggID, "CounterCreated")
	}
}

func BenchmarkDecider_Execute_Update(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)
	aggID := id.NewAggregateID()

	seedCounterBench(b, repo, aggID, 100)

	b.ResetTimer()

	for b.Loop() {
		benchExecute(b, repo, ctx, aggID, "CounterIncremented")
	}
}

func BenchmarkDecider_Load(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)
	aggID := id.NewAggregateID()

	seedCounterBench(b, repo, aggID, 100)

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, aggID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkDecider_Apply(b *testing.B) {
	b.ReportAllocs()
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
			s, _ = applyCounter(s, evt)
		}
	}
}
