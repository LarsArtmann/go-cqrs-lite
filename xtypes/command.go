package xtypes

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedCommand wraps a command with a strongly-typed aggregate ID.
type TypedCommand[A any] struct {
	commandType command.Type
	aggregateID id.Of[A]
}

// Type returns the command type.
func (c *TypedCommand[A]) Type() command.Type {
	return c.commandType
}

// AggregateID returns the strongly-typed aggregate ID.
func (c *TypedCommand[A]) AggregateID() id.Of[A] {
	return c.aggregateID
}

// Command returns the underlying command.Command interface.
func (c *TypedCommand[A]) Command() command.Command {
	return command.New(c.commandType, c.aggregateID.String())
}

// NewTypedCommand creates a new typed command.
func NewTypedCommand[A any](
	commandType command.Type,
	aggregateID id.Of[A],
) *TypedCommand[A] {
	return &TypedCommand[A]{
		commandType: commandType,
		aggregateID: aggregateID,
	}
}
