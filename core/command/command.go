package command

import (
	"fmt"

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

// New creates a new command with validation.
func New(commandType Type, aggregateID id.AggregateID) (*Core, error) {
	if commandType == "" {
		//nolint:err113 // dynamic error required to include aggregate ID
		return nil, fmt.Errorf(
			"command type is required (got empty) for aggregate %q",
			aggregateID.String(),
		)
	}

	if aggregateID.IsZero() {
		//nolint:err113 // dynamic error required to include command type
		return nil, fmt.Errorf(
			"aggregate ID is required (got zero) for command type %q",
			commandType,
		)
	}

	return &Core{
		commandType: commandType,
		aggregateID: aggregateID,
	}, nil
}

// MustNew creates a new command or panics on validation failure.
// Use only in tests where inputs are guaranteed valid.
func MustNew(commandType Type, aggregateID id.AggregateID) *Core {
	c, err := New(commandType, aggregateID)
	if err != nil {
		panic(fmt.Sprintf("command.MustNew: %v", err))
	}

	return c
}
