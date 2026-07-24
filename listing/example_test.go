package listing_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func ExampleNewInMemoryStreamReader() {
	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryStreamReader(store)

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)

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

func ExampleNewListBuilder() {
	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryStreamReader(store)

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Order", aggID)

	evt, _ := event.NewEvent("OrderPlaced", aggID, "Order", 1, []byte(`{"total":42}`))
	_ = store.Save(context.Background(), ref, []event.Event{evt}, 0)

	page, err := listing.NewListBuilder(reader).
		OfType("Order").
		PageSize(10).
		List(context.Background())
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(len(page.Items), page.HasMore)

	// Output:
	// 1 false
}

func ExampleStatusMiddleware() {
	bus := eventtest.NewFakeBus()

	deleteTypes := []event.Type{"user.deleted"}
	rebirthTypes := []event.Type{"user.restored"}

	_ = bus.UsePublish(listing.StatusMiddleware(deleteTypes, rebirthTypes))

	fmt.Println("StatusMiddleware installed")

	// Output:
	// StatusMiddleware installed
}

func ExampleCacheInvalidationMiddleware() {
	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryStreamReader(store)
	bus := eventtest.NewFakeBus()

	// Invalidate reader cache whenever events are published
	_ = bus.UsePublish(listing.CacheInvalidationMiddleware(reader))

	// Seed a stream
	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	_ = store.Save(context.Background(), ref, []event.Event{evt}, 0)

	// First read populates the cache
	page, _ := reader.List(context.Background(), listing.ListOptions{Type: "User"})
	fmt.Println("before publish:", len(page.Items))

	// Publishing through the bus invalidates the cache
	_ = bus.Publish(context.Background(), evt)

	// Second read reflects the invalidation
	page, _ = reader.List(context.Background(), listing.ListOptions{Type: "User"})
	fmt.Println("after publish:", len(page.Items))

	// Output:
	// before publish: 1
	// after publish: 1
}
