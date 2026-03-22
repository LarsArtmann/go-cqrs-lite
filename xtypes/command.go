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

// CommandBuilder constructs commands with compile-time type safety.
type CommandBuilder[A any] struct {
    commandType command.Type
    aggregateID id.Of[A]
}

// NewCommandBuilder creates a new command builder.
func NewCommandBuilder[A any](
    commandType command.Type,
    aggregateID id.Of[A],
) *CommandBuilder[A] {
    return &CommandBuilder[A]{
        commandType:    commandType,
        aggregateID:   aggregateID,
    }
}

// Build creates the typed command.
func (b *CommandBuilder[A]) Build() *TypedCommand[A] {
    if b.aggregateID.IsEmpty() {
        return nil, fmt.Errorf("aggregate ID is required for command type %q", b.commandType)
    }
    return &TypedCommand[A]{
        commandType:   b.commandType,
        aggregateID: b.aggregateID,
    }, nil
}
