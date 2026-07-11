package schema_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

func ExampleNewUpcaster() {
	upcaster := schema.NewUpcaster(
		"UserCreated",
		1,
		func(evt event.Event) (event.Event, error) {
			return event.NewEvent(
				evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
				[]byte(`{"name":"unknown","email":"pending"}`),
				event.WithSchemaVersion(2),
			)
		},
	)

	fmt.Println(upcaster.SourceType())
	fmt.Println(upcaster.SourceVersion())

	// Output:
	// UserCreated
	// 1
}
