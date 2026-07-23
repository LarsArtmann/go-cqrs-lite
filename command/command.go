package command

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Type identifies a command type.
type Type string

// String returns the command type as a string.
func (t Type) String() string { return string(t) }

// IsZero returns true if the command type is empty.
func (t Type) IsZero() bool { return t == "" }

// ParseType validates and returns a Type. Returns an error if empty.
func ParseType(s string) (Type, error) {
	if s == "" {
		return "", ErrEmptyCommandType
	}

	return Type(s), nil
}

// Command represents a domain command.
//
// Every command carries an [id.CommandID], minted at construction time via
// [New]. The ID is stable for the lifetime of the command object — retrying
// the same logical command with a new [New] call produces a new ID, so
// consumers needing idempotency across retries should pass [WithCommandID]
// with a deterministic key.
type Command interface {
	Type() Type
	StreamID() id.StreamID
	ID() id.CommandID
}

// BasicCommand provides a default implementation.
type BasicCommand struct {
	commandID   id.CommandID
	commandType Type
	streamID id.StreamID
	metadata    Metadata
}

var _ Command = (*BasicCommand)(nil)

// Type returns the command type.
func (c *BasicCommand) Type() Type { return c.commandType }

// StreamID returns the aggregate ID.
func (c *BasicCommand) StreamID() id.StreamID { return c.streamID }

// ID returns the command ID, minted at construction time.
func (c *BasicCommand) ID() id.CommandID { return c.commandID }

// Metadata returns the command metadata.
func (c *BasicCommand) Metadata() Metadata { return c.metadata.Clone() }

// New creates a new command with validation.
func New(commandType Type, streamID id.StreamID, opts ...Option) (*BasicCommand, error) {
	if commandType == "" {
		return nil, errorfamily.WrapRejection(
			ErrEmptyCommandType,
			"command.empty_command_type",
			"command type is required: got empty for aggregate "+streamID.String(),
		)
	}

	if streamID.IsZero() {
		return nil, errorfamily.WrapRejection(
			ErrNilAggregateID,
			"command.nil_aggregate_id",
			"aggregate ID is required: got zero for command type "+string(commandType),
		)
	}

	cmd := &BasicCommand{
		commandID:   id.NewCommandID(),
		commandType: commandType,
		streamID: streamID,
		metadata:    Metadata{}, //nolint:exhaustruct // zero-value metadata is the correct initial state
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd, nil
}
