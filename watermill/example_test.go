package watermill_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

func ExampleNewSubscriberAdapter() {
	bus := eventtest.NewFakeBus()
	adapter := watermill.NewSubscriberAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}

func ExampleNewPublisherAdapter() {
	bus := eventtest.NewFakeBus()
	adapter := watermill.NewPublisherAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}
