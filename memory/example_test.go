package memory_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func ExampleNewMemoryStore() {
	store := memory.NewMemoryStore()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))

	err := store.Save(context.Background(), ref, []event.Event{evt}, 0)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	events, _ := store.Load(context.Background(), ref)
	fmt.Println(len(events))

	// Output:
	// 1
}
