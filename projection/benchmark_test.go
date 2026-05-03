package projection_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func benchEvent(tb testing.TB, eventType string, aggID id.AggregateID, version int) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "User", version, nil)
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func BenchmarkRunner_Replay(b *testing.B) {
	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	events := make([][]event.Event, 10)

	for i := range 10 {
		aggID := id.NewAggregateID()
		events[i] = []event.Event{benchEvent(b, "UserCreated", aggID, 1)}

		err := store.Save(ctx, "User", aggID, events[i], 0)
		if err != nil {
			b.Fatalf("Save: %v", err)
		}
	}

	for b.Loop() {
		bus := memory.NewMemoryBus()
		b.Cleanup(func() { _ = bus.Close() })

		checkpoint := memory.NewCheckpointStore()
		b.Cleanup(func() { _ = checkpoint.Close() })

		runner, err := projection.NewRunner(store, bus, checkpoint)
		if err != nil {
			b.Fatalf("NewRunner: %v", err)
		}

		err = runner.Register(event.NewProjection("bench-proj",
			func(_ context.Context, _ event.Event) error { return nil },
			[]event.Type{"UserCreated"},
		))
		if err != nil {
			b.Fatalf("Register: %v", err)
		}

		runCtx, cancel := context.WithCancel(ctx)

		done := make(chan struct{})

		go func() {
			_ = runner.Run(runCtx)
			close(done)
		}()

		<-done

		cancel()
	}
}

func BenchmarkRunner_FilterEvents(b *testing.B) {
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	err = runner.Register(event.NewProjection("bench-proj",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	aggID := id.NewAggregateID()

	for b.Loop() {
		_ = runner.Register(event.NewProjection("noop-proj",
			func(_ context.Context, _ event.Event) error { return nil },
			[]event.Type{"UserCreated"},
		))
	}

	_ = aggID
}
