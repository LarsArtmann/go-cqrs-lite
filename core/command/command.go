package command

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Type identifies a command type.
type Type string

// String returns the command type as a string.
func (t Type) String() string { return string(t) }

// Command represents a domain command.
type Command interface {
	Type() Type
	AggregateID() id.AggregateID

	// Deprecated: Idempotency handling should be managed at the application layer,
	// not embedded in the command interface. This method will be removed in a future version.
	IdempotencyKey() string
}

// Core provides a default implementation.
type Core struct {
	commandType Type
	aggregateID id.AggregateID
}

var _ Command = (*Core)(nil)

// Type returns the command type.
func (c *Core) Type() Type { return c.commandType }

// AggregateID returns the aggregate ID.
func (c *Core) AggregateID() id.AggregateID { return c.aggregateID }

// IdempotencyKey returns a deduplication key for the command.
// Returns empty string by default — consumers should override for production use.
func (c *Core) IdempotencyKey() string { return "" }

// New creates a new command with validation.
func New(commandType Type, aggregateID id.AggregateID) (*Core, error) {
	if commandType == "" {
		return nil, fmt.Errorf(
			"%w: got empty for aggregate %q",
			ErrEmptyCommandType,
			aggregateID,
		)
	}

	if aggregateID.IsZero() {
		return nil, fmt.Errorf(
			"%w: got zero for command type %q",
			ErrNilAggregateID,
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
