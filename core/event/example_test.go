package event_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func ExampleNewEvent() {
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent(
		"UserCreated",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
		event.WithCorrelationID(id.NewCorrelationID()),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(evt.Type())
	fmt.Println(evt.Version())

	// Output:
	// UserCreated
	// 1
}

func ExampleNewBuilder() {
	aggID := id.NewAggregateID()

	evt := event.NewBuilder("UserCreated", aggID, "User", 1).
		WithPayload([]byte(`{"name":"Alice"}`)).
		MustBuild()

	fmt.Println(evt.Type())

	// Output:
	// UserCreated
}

func ExampleInMemoryRunner() {
	checkpoint := memory.NewCheckpointStore()
	runner := event.NewInMemoryRunner(checkpoint)

	proj := event.NewProjection(
		"my-projection",
		func(_ context.Context, evt event.Event) error {
			fmt.Println(string(evt.Type()))

			return nil
		},
		[]event.Type{"UserCreated"},
	)

	_ = runner.Register(proj)

	evt, _ := event.NewEvent("UserCreated", id.NewAggregateID(), "User", 1, nil)
	_ = runner.Handle(context.Background(), evt)

	// Output:
	// UserCreated
}
