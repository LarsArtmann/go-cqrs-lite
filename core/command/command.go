package command

import (
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Type identifies a command type.
type Type string

// Command represents a domain command.
type Command interface {
	Type() Type
	AggregateID() id.AggregateID
}

// Core provides a default implementation.
type Core struct {
	commandType Type
	aggregateID id.AggregateID
}

// Type returns the command type.
func (c *Core) Type() Type { return c.commandType }

// AggregateID returns the aggregate ID.
func (c *Core) AggregateID() id.AggregateID { return c.aggregateID }

// New creates a new command.
func New(commandType Type, aggregateID id.AggregateID) *Core {
	return &Core{
		commandType: commandType,
		aggregateID: aggregateID,
	}
}
