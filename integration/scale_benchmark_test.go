package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type benchState struct{ Value int }

func benchApply(_ benchState, evt event.Event) (benchState, error) {
	switch evt.Type() {
	case "ItemCreated":
		return benchState{Value: 1}, nil
	case "ItemUpdated":
		return benchState{Value: 2}, nil
	}

	return benchState{}, nil
}

func benchDecider() decider.Decider[benchState] {
	return decider.Decider[benchState]{Initial: benchState{}, Apply: benchApply}
}

func newBenchDeciderRepo(b *testing.B) (*decider.Repository[benchState], context.Context) {
	b.Helper()

	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	repo, err := decider.NewRepository(store, bus, benchDecider())
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	return repo, context.Background()
}

func newBenchEvent(
	b *testing.B,
	eventType string,
	streamID id.StreamID,
	v event.Version,
) event.Event {
	b.Helper()

	evt, err := event.NewEvent(event.Type(eventType), streamID, "Item", v, nil)
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func noopCmdHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error { return nil }
}

func benchNoopQueryHandler(_ context.Context, _ query.Query) (any, error) {
	return nil, nil
}

func noopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error { return nil }
}

func benchCreateItem(
	b *testing.B,
	repo *decider.Repository[benchState],
	ctx context.Context,
	streamID id.StreamID,
) {
	b.Helper()

	err := repo.Execute(
		ctx, streamID, "Item",
		func(_ benchState, v event.Version) ([]event.Event, error) {
			evt := newBenchEvent(b, "ItemCreated", streamID, v.Increment())

			return []event.Event{evt}, nil
		},
	)
	if err != nil {
		b.Fatalf("Execute: %v", err)
	}
}

func benchCreateItemConcurrent(
	b *testing.B,
	repo *decider.Repository[benchState],
	ctx context.Context,
) id.StreamID {
	b.Helper()

	streamID := id.NewStreamID()
	decideFn := func(_ benchState, v event.Version) ([]event.Event, error) {
		return []event.Event{newBenchEvent(b, "ItemCreated", streamID, v.Increment())}, nil
	}

	if err := repo.Execute(ctx, streamID, "Item", decideFn); err != nil {
		b.Errorf("concurrent Execute: %v", err)
	}

	return streamID
}

// ---------------------------------------------------------------------------
// 1. Million Commands — sustained dispatch throughput
// ---------------------------------------------------------------------------

func BenchmarkScale_CommandDispatch(b *testing.B) {
	b.ReportAllocs()

	dispatcher := command.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	for i := range 100 {
		err := dispatcher.Register(
			command.Type(fmt.Sprintf("cmd.%d", i)),
			noopCmdHandler(),
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	streamID := id.NewStreamID()
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for i := range 100 {
			cmd := testutil.NewCmd(b, command.Type(fmt.Sprintf("cmd.%d", i)), streamID)
			err := dispatcher.Dispatch(ctx, cmd)
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "commands/sec")
}

// ---------------------------------------------------------------------------
// 2. Million Events — creation + save + publish at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_EventCreation(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()

	b.ResetTimer()

	for b.Loop() {
		evt, err := event.NewEvent("ItemCreated", streamID, "Item", 1, nil)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
		if evt == nil {
			b.Fatal("NewEvent returned nil")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}
