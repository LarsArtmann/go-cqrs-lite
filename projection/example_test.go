package projection_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

type userCreatedPayload struct {
	Name string `json:"name"`
}

func ExampleNewBuilder() {
	b := projection.NewBuilder("user-projection")

	_ = projection.On[userCreatedPayload](
		b,
		"user.created",
		codec.JSONCodec{},
		func(_ context.Context, _ userCreatedPayload) error {
			return nil
		},
	)

	p := b.Build()

	fmt.Println(p.Name())

	// Output:
	// user-projection
}
