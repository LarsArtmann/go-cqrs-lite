package command

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Type identifies a command type.
//
// It is an alias of record.Type (ADR-0111): one definition shared with
// event and query, so the per-module copies cannot drift.
type Type = record.Type

// ParseType validates and returns a Type. Returns an error if empty.
//
// Deprecated: removed in v5. Use record.ParseType(s, ErrEmptyCommandType).
func ParseType(s string) (Type, error) {
	return record.ParseType(s, ErrEmptyCommandType)
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
	streamID    id.StreamID
	metadata    Metadata
}

var _ Command = (*BasicCommand)(nil)

// Type returns the command type.
func (c *BasicCommand) Type() Type { return c.commandType }

// StreamID returns the stream ID.
func (c *BasicCommand) StreamID() id.StreamID { return c.streamID }

// ID returns the command ID, minted at construction time.
// cqrs-lint:ignore(A001) library code or intentional pattern
func (c *BasicCommand) ID() id.CommandID { return c.commandID }

// Metadata returns the command metadata.
func (c *BasicCommand) Metadata() Metadata { return c.metadata.Clone() }

// ApplyOptions applies metadata options to an already-constructed command.
// Intended for pipeline enrichment: transport adapters inject request-scoped
// metadata (actor IDs, correlation IDs) after the domain decoder creates the
// command but before dispatch. Options that set already-populated fields
// will overwrite them.
func (c *BasicCommand) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// New creates a new command with validation.
func New(commandType Type, streamID id.StreamID, opts ...Option) (*BasicCommand, error) {
	if commandType == "" {
		return nil, errorfamily.WrapRejection(
			ErrEmptyCommandType,
			"command.empty_command_type",
			"command type is required: got empty for stream "+streamID.String(),
		)
	}

	if streamID.IsZero() {
		return nil, errorfamily.WrapRejection(
			ErrNilStreamID,
			"command.nil_aggregate_id",
			"stream ID is required: got zero for command type "+string(commandType),
		)
	}

	cmd := &BasicCommand{
		commandID:   id.NewCommandID(),
		commandType: commandType,
		streamID:    streamID,
		metadata:    Metadata{}, //nolint:exhaustruct_v5 // zero-value metadata is the correct initial state
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd, nil
}
