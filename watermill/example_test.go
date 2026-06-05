package watermill_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

func ExampleNewSubscriberAdapter() {
	bus := memory.NewMemoryBus()
	adapter := watermill.NewSubscriberAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}

func ExampleNewPublisherAdapter() {
	bus := memory.NewMemoryBus()
	adapter := watermill.NewPublisherAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}
