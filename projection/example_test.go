package projection_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func ExampleNewBuilder() {
	b := projection.NewBuilder("user-projection").
		On("user.created", func(_ context.Context, _ event.Event) error {
			return nil
		})

	fmt.Println(b != nil)

	// Output:
	// true
}
