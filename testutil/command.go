package testutil

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func NewCmd(
	tb testing.TB,
	commandType command.Type,
	aggregateID id.AggregateID,
	opts ...command.Option,
) *command.BasicCommand {
	tb.Helper()

	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		tb.Fatalf("testutil: new command %q: %v", commandType, err)
	}

	return cmd
}

func NoopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}
