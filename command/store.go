package command

import (
	"context"
	"fmt"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type PersistedCommand struct {
	id         id.CommandID
	cmdType    Type
	streamRef  StreamRef
	receivedAt time.Time
	payload    []byte
	metadata   Metadata
}

var (
	_ fmt.Stringer = (*PersistedCommand)(nil)
	_ Command      = (*PersistedCommand)(nil)
)

//cqrs-lint:ignore(A001,E005) library code or intentional pattern
func (c *PersistedCommand) ID() id.CommandID       { return c.id }
func (c *PersistedCommand) Type() Type             { return c.cmdType }
func (c *PersistedCommand) StreamID() id.StreamID  { return c.streamRef.ID }
func (c *PersistedCommand) StreamType() StreamType { return c.streamRef.Type }
func (c *PersistedCommand) StreamRef() StreamRef   { return c.streamRef }
func (c *PersistedCommand) ReceivedAt() time.Time  { return c.receivedAt }
func (c *PersistedCommand) Payload() []byte {
	if c.payload == nil {
		return nil
	}

	return slices.Clone(c.payload)
}
func (c *PersistedCommand) Metadata() Metadata { return c.metadata.Clone() }

func (c *PersistedCommand) String() string {
	return fmt.Sprintf("%s(%s) %s@%s",
		c.cmdType, c.id, c.streamRef.Type, c.streamRef.ID)
}

type PersistOption func(*PersistedCommand)

func WithReceivedAt(t time.Time) PersistOption {
	return func(c *PersistedCommand) { c.receivedAt = t }
}

func WithPersistedCommandID(cmdID id.CommandID) PersistOption {
	return func(c *PersistedCommand) { c.id = cmdID }
}

func WithCommandMetadata(m Metadata) PersistOption {
	return func(c *PersistedCommand) { c.metadata = m.Clone() }
}

func NewPersistedCommand(
	cmdType Type,
	ref StreamRef,
	payload []byte,
	opts ...PersistOption,
) (*PersistedCommand, error) {
	if cmdType == "" {
		return nil, errorfamily.WrapRejection(
			ErrEmptyCommandType,
			"command.empty_command_type",
			"command type is required",
		)
	}

	if ref.Type.IsZero() {
		return nil, errorfamily.WrapRejection(
			ErrEmptyStreamType,
			"command.empty_aggregate_type",
			"stream type is required in ref",
		)
	}

	if ref.ID.IsZero() {
		return nil, errorfamily.WrapRejection(
			ErrNilStreamID,
			"command.nil_aggregate_id",
			"stream ID is required in ref",
		)
	}

	cmd := &PersistedCommand{
		id:         id.NewCommandID(),
		cmdType:    cmdType,
		streamRef:  ref,
		receivedAt: time.Now(),
		payload:    slices.Clone(payload),
		metadata:   Metadata{}, //nolint:exhaustruct // zero-value metadata is the correct initial state
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd, nil
}

type CommandSink interface {
	Save(ctx context.Context, ref StreamRef, cmd *PersistedCommand) error

	AppendBatch(ctx context.Context, ref StreamRef, cmds []*PersistedCommand) error
}

type CommandSource interface {
	Load(ctx context.Context, ref StreamRef) ([]*PersistedCommand, error)

	LoadFromTimestamp(
		ctx context.Context,
		ref StreamRef,
		after time.Time,
	) ([]*PersistedCommand, error)

	LoadToTimestamp(
		ctx context.Context,
		ref StreamRef,
		maxTime time.Time,
	) ([]*PersistedCommand, error)
}

type Store interface {
	CommandSink
	CommandSource
}

// CommandJournal reads all commands across all streams, ordered by
// ReceivedAt. This is the command-side equivalent of event.Journal —
// it provides a complete audit trail of every command ever dispatched.
//
// Use cases: audit ("who issued what commands and when?"), replay
// debugging, analytics ("which command types are most frequent?").
type CommandJournal interface {
	ReadAll(ctx context.Context) ([]*PersistedCommand, error)
}

// SeekableCommandJournal extends CommandJournal with position-based reading.
// Position is based on CommandID (ULID-based, time-sortable).
//
// Enables incremental command replay: read commands in batches from a
// checkpoint, process them, then resume from the last CommandID.
type SeekableCommandJournal interface {
	CommandJournal
	ReadFrom(
		ctx context.Context,
		afterCommandID id.CommandID,
		limit int,
	) ([]*PersistedCommand, error)
}
