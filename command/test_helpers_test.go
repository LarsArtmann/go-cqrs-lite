package command_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func mustNewCmd(
	commandType command.Type,
	aggregateID id.AggregateID,
	opts ...command.Option,
) *command.BasicCommand {
	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		panic(err)
	}

	return cmd
}

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
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
