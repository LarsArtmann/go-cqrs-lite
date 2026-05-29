package event_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
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
	_ = store.Save(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{v1Event},
		0,
	)

	upcaster := makeUpcaster(
		"UserCreated",
		1,
		[]byte(`{"name":"Alice","fullname":"Alice Wonderland"}`),
	)

	versioned := event.NewVersionedStore(store, upcaster)

	events, err := versioned.Load(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), aggID),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(events[0].SchemaVersion())

	// Output:
	// 2
}
