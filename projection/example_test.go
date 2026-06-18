package projection_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

type userCreatedPayload struct {
	Name string `json:"name"`
}

func ExampleNewBuilder() {
	b := projection.NewBuilder("user-projection")

	_ = projection.On[userCreatedPayload](
		b,
		"user.created",
		codec.JSONCodec{},
		func(_ context.Context, _ userCreatedPayload) error {
			return nil
		},
	)

	p := b.Build()

	fmt.Println(p.Name())

	// Output:
	// user-projection
}

// ExampleRunner demonstrates the read-your-writes projection pattern: RunReplay
// catches up synchronously, then RunLive tails live events in the background.
// This eliminates time.Sleep-based catch-up hacks in consumers.
func ExampleRunner() {
	store := memory.NewMemoryStore()
	defer func() { _ = store.Close() }()

	bus := memory.NewMemoryBus()
	defer func() { _ = bus.Close() }()

	checkpoint := memory.NewMemoryCheckpointStore()
	defer func() { _ = checkpoint.Close() }()

	// Seed historical events before the runner starts.
	aggID := id.NewAggregateID()
	ctx := context.Background()
	historicalEvent, _ := event.NewEvent(
		event.Type("UserCreated"), aggID, "User", 1, nil,
	)
	_ = store.Save(ctx, event.NewAggregateRef("User", aggID), []event.Event{
		historicalEvent,
	}, 0)

	runner, _ := projection.NewRunner(store, bus, checkpoint)

	// Register a projection that builds a read model.
	var count int
	_ = runner.Register(event.NewProjection(
		"user-count",
		func(_ context.Context, _ event.Event) error {
			count++
			return nil
		},
		[]event.Type{"UserCreated"},
	))

	// RunReplay blocks until all historical events are processed.
	// After this returns, the read model is caught up — no sleep needed.
	_ = runner.RunReplay(ctx)
	fmt.Println("caught up:", count)

	// RunLive tails live events in the background. Cancel to stop.
	liveCtx, cancel := context.WithCancel(ctx)
	go func() { _ = runner.RunLive(liveCtx) }()

	// ... serve requests against the read model here ...

	cancel() // stop the live tail
	_ = runner.Close()

	// Output: caught up: 1
}
