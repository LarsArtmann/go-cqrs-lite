package command

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
	aggregateID string
}

func (c *BaseCommand) Type() Type          { return c.commandType }
func (c *BaseCommand) AggregateID() string { return c.aggregateID }

// New creates a new command
func New(commandType Type, aggregateID string) *BaseCommand {
	return &BaseCommand{
		commandType: commandType,
		aggregateID: aggregateID,
	}
}
