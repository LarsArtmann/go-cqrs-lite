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

func ExampleInMemoryRunner() {
	checkpoint := memory.NewMemoryCheckpointStore()

	runner, err := event.NewInMemoryRunner(checkpoint)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

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

func ExampleNewVersionedStore() {
	aggID := id.NewAggregateID()

	v1Event, _ := event.NewEvent(
		"UserCreated",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice","fullname":""}`),
		event.WithSchemaVersion(1),
	)

	store := memory.NewMemoryStore()
	_ = store.Save(context.Background(), "User", aggID, []event.Event{v1Event}, 0)

	upcaster := event.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
		return event.NewEvent(
			evt.Type(),
			evt.AggregateID(),
			evt.AggregateType(),
			evt.Version(),
			[]byte(`{"name":"Alice","fullname":"Alice Wonderland"}`),
			event.WithSchemaVersion(2),
		)
	})

	versioned := event.NewVersionedStore(store, upcaster)

	events, err := versioned.Load(context.Background(), "User", aggID)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(events[0].SchemaVersion())

	// Output:
	// 2
}
