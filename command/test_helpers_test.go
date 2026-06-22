package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func newCmd(
	tb testing.TB,
	commandType command.Type,
	aggregateID id.AggregateID,
	opts ...command.Option,
) *command.BasicCommand {
	tb.Helper()

	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		tb.Fatalf("new command %q: %v", commandType, err)
	}

	return cmd
}

func noopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

func callbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

func appendCommandHandler(callOrder *[]string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*callOrder = append(*callOrder, "handler")

		return nil
	}
}
