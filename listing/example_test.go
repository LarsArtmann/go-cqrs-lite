package listing_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func ExampleNewInMemoryAggregateReader() {
	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryAggregateReader(store)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))
	_ = store.Save(context.Background(), ref, []event.Event{evt}, 0)

	page, err := reader.ListWithStatus(context.Background(), listing.ListOptions{
		Type: "User",
	})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(len(page.Items))

	// Output:
	// 1
}
