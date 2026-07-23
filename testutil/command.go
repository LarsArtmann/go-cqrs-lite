package testutil

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func NewCmd(
	tb testing.TB,
	commandType command.Type,
	streamID id.StreamID,
	opts ...command.Option,
) *command.BasicCommand {
	tb.Helper()

	cmd, err := command.New(commandType, streamID, opts...)
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
