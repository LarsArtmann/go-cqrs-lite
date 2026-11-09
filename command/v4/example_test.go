package command_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
