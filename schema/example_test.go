package schema_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/schema/v2"
)

func ExampleNewUpcaster() {
	upcaster, err := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
		return event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			map[string]any{"name": "unknown", "email": "pending"},
			event.WithSchemaVersion(2),
		)
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(upcaster.SourceType())
	fmt.Println(upcaster.SourceVersion())

	// Output:
	// UserCreated
	// 1
}
