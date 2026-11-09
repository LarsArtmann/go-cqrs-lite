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

	var lastStream id.StreamID

	for b.Loop() {
		streamID := id.NewStreamID()
		lastStream = streamID
		benchExecute(b, repo, ctx, streamID, "CounterCreated")
	}

	// Verify events were persisted (not silently dropped).
	state, _, err := repo.Load(ctx, lastStream, "Counter")
	if err != nil {
		b.Fatalf("post-loop Load: %v", err)
	}
	if state.Value != 1 {
		b.Fatalf("post-loop Load: state.Value=%d, want 1 — Execute was a no-op", state.Value)
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

	// Verify the increment events were persisted.
	state, _, err := repo.Load(ctx, streamID, "Counter")
	if err != nil {
		b.Fatalf("post-loop Load: %v", err)
	}
	if state.Value < 101 {
		b.Fatalf("post-loop Load: state.Value=%d, want >=101 — increments were no-ops", state.Value)
	}
}

func BenchmarkDecider_Load(b *testing.B) {
	b.ReportAllocs()
	repo, ctx := newBenchRepo(b)
	streamID := id.NewStreamID()

	seedCounterBench(b, repo, streamID, 100)

	b.ResetTimer()

	for b.Loop() {
		state, _, err := repo.Load(ctx, streamID, "Counter")
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
		if state.Value != 100 {
			b.Fatalf("Load: state.Value=%d, want 100 — fold produced wrong state", state.Value)
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
			var err error
			s, err = applyCounter(s, evt)
			if err != nil {
				b.Fatalf("applyCounter: %v", err)
			}
		}

		if s.Value != 100 {
			b.Fatalf("after folding 100 increments: state.Value=%d, want 100", s.Value)
		}
	}
}
