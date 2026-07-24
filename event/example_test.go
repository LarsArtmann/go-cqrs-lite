package event_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func ExampleNewEvent() {
	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		"UserCreated",
		streamID,
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
	streamID := id.NewStreamID()

	v1Event, _ := event.NewEvent(
		"UserCreated",
		streamID,
		"User",
		1,
		[]byte(`{"name":"Alice","fullname":""}`),
		event.WithSchemaVersion(1),
	)

	store := memory.NewMemoryStore()
	_ = store.Save(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{v1Event},
		0,
	)

	upcaster := makeUpcaster(
		"UserCreated",
		1,
		[]byte(`{"name":"Alice","fullname":"Alice Wonderland"}`),
	)

	versioned, err := schema.NewVersionedStore(store, upcaster)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	events, err := versioned.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), streamID),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(events[0].SchemaVersion())

	// Output:
	// 2
}
