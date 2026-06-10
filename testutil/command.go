// Package testutil provides shared test helpers for go-cqrs-lite.
package testutil

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// MustNewCmd creates a BasicCommand and panics on error.
func MustNewCmd(
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

// ParseAggID parses an AggregateID and panics on error.
func ParseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}

// NoopCommandHandler returns a handler that does nothing.
func NoopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}
