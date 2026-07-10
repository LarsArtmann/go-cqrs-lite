package memory_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

func ExampleNewMemoryStore() {
	store := memory.NewMemoryStore()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("User", aggID)

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
