package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func benchEvent(
	tb testing.TB,
	eventType string,
	streamID id.StreamID,
	version event.Version,
) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(event.Type(eventType), streamID, "Counter", version, []byte("{}"))
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func seedCounterBench(
	b *testing.B,
	repo *decider.Repository[counterState],
	streamID id.StreamID,
	n int,
) {
	b.Helper()

	ctx := context.Background()

	for range n {
		benchExecute(b, repo, ctx, streamID, "CounterIncremented")
	}
}

func benchExecute(
	b *testing.B,
	repo *decider.Repository[counterState],
	ctx context.Context,
	streamID id.StreamID,
	eventType string,
) {
	b.Helper()

	err := repo.Execute(
		ctx, streamID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{benchEvent(b, eventType, streamID, v.Increment())}, nil
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
		streamID := id.NewStreamID()
		benchExecute(b, repo, ctx, streamID, "CounterCreated")
	}
}

func BenchmarkDecider_Execute_Update(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)
	streamID := id.NewStreamID()

	seedCounterBench(b, repo, streamID, 100)

	b.ResetTimer()

	for b.Loop() {
		benchExecute(b, repo, ctx, streamID, "CounterIncremented")
	}
}

func BenchmarkDecider_Load(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)
	streamID := id.NewStreamID()

	seedCounterBench(b, repo, streamID, 100)

	b.ResetTimer()

	for b.Loop() {
		_, _, err := repo.Load(ctx, streamID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkDecider_Apply(b *testing.B) {
	b.ReportAllocs()
	events := make([]event.Event, 100)
	streamID := id.NewStreamID()

	for i := range 100 {
		events[i] = benchEvent(b, "CounterIncremented", streamID, event.Version(i+1))
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
