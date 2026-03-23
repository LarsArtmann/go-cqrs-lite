package command

import (
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Type identifies a command type
type Type string

// Command represents a domain command
type Command interface {
	Type() Type
	AggregateID() string
}

// BaseCommand provides a default implementation
type BaseCommand struct {
	commandType Type
	aggregateID id.AggregateID
}

func (c *BaseCommand) Type() Type          { return c.commandType }
func (c *BaseCommand) AggregateID() string { return c.aggregateID.String() }

// New creates a new command
func New(commandType Type, aggregateID id.AggregateID) *BaseCommand {
	return &BaseCommand{
		commandType: commandType,
		aggregateID: aggregateID,
	}
}
