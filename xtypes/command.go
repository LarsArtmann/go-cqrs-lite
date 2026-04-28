package xtypes

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// TypedCommand wraps a command with a strongly-typed aggregate ID.
type TypedCommand struct {
	commandType command.Type
	aggregateID id.AggregateID
}

// Type returns the command type.
func (c *TypedCommand) Type() command.Type {
	return c.commandType
}

// AggregateID returns the strongly-typed aggregate ID.
func (c *TypedCommand) AggregateID() id.AggregateID {
	return c.aggregateID
}

// Command returns the underlying command.Command interface.
func (c *TypedCommand) Command() (command.Command, error) {
	return command.New(c.commandType, c.aggregateID) //nolint:wrapcheck // same monorepo
}

// MustCommand returns the underlying command.Command or panics.
func (c *TypedCommand) MustCommand() command.Command {
	cmd, err := c.Command()
	if err != nil {
		panic(fmt.Sprintf("TypedCommand.MustCommand: %v", err))
	}

	return cmd
}

// NewTypedCommand creates a new typed command.
func NewTypedCommand(
	commandType command.Type,
	aggregateID id.AggregateID,
) *TypedCommand {
	return &TypedCommand{
		commandType: commandType,
		aggregateID: aggregateID,
	}
}
