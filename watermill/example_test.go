package watermill_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

func ExampleNewSubscriberAdapter() {
	fmt.Println(watermill.NewSubscriberAdapter != nil)

	// Output:
	// true
}

func ExampleNewPublisherAdapter() {
	fmt.Println(watermill.NewPublisherAdapter != nil)

	// Output:
	// true
}
