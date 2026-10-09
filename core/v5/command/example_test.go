package command_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/v5/command"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
)

func ExampleDispatcher() {
	d := command.NewDispatcher()

	_ = d.Register("CreateUser", func(_ context.Context, _ command.Command) error {
		fmt.Println("handled")

		return nil
	})

	cmd, _ := command.New("CreateUser", id.NewStreamID())
	_ = d.Dispatch(context.Background(), cmd)

	// Output:
	// handled
}
