package command

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type PersistedCommand struct {
	id           id.CommandID
	cmdType      Type
	aggregateRef AggregateRef
	receivedAt   time.Time
	payload      []byte
	metadata     Metadata
}

var _ fmt.Stringer = (*PersistedCommand)(nil)

func (c *PersistedCommand) ID() id.CommandID             { return c.id }
func (c *PersistedCommand) Type() Type                   { return c.cmdType }
func (c *PersistedCommand) AggregateID() id.AggregateID  { return c.aggregateRef.ID }
func (c *PersistedCommand) AggregateType() AggregateType { return c.aggregateRef.Type }
func (c *PersistedCommand) AggregateRef() AggregateRef   { return c.aggregateRef }
func (c *PersistedCommand) ReceivedAt() time.Time        { return c.receivedAt }
func (c *PersistedCommand) Payload() []byte {
	if c.payload == nil {
		return nil
	}

	return slices.Clone(c.payload)
}
func (c *PersistedCommand) Metadata() Metadata { return c.metadata.Clone() }

func (c *PersistedCommand) String() string {
	return fmt.Sprintf("%s(%s) %s@%s",
		c.cmdType, c.id, c.aggregateRef.Type, c.aggregateRef.ID)
}

type PersistOption func(*PersistedCommand)

func WithReceivedAt(t time.Time) PersistOption {
	return func(c *PersistedCommand) { c.receivedAt = t }
}

func WithCommandID(cmdID id.CommandID) PersistOption {
	return func(c *PersistedCommand) { c.id = cmdID }
}

func WithCommandMetadata(m Metadata) PersistOption {
	return func(c *PersistedCommand) { c.metadata = m.Clone() }
}

func NewPersistedCommand(
	cmdType Type,
	ref AggregateRef,
	payload []byte,
	opts ...PersistOption,
) (*PersistedCommand, error) {
	if cmdType == "" {
		return nil, event.WrapRejection(
			ErrEmptyCommandType,
			"command.empty_command_type",
			"command type is required",
		)
	}

	if ref.Type.IsZero() {
		return nil, event.WrapRejection(
			ErrEmptyAggregateType,
			"command.empty_aggregate_type",
			"aggregate type is required in ref",
		)
	}

	if ref.ID.IsZero() {
		return nil, event.WrapRejection(
			ErrNilAggregateID,
			"command.nil_aggregate_id",
			"aggregate ID is required in ref",
		)
	}

	cmd := &PersistedCommand{
		id:           id.NewCommandID(),
		cmdType:      cmdType,
		aggregateRef: ref,
		receivedAt:   time.Now(),
		payload:      slices.Clone(payload),
		metadata:     NewMetadata(),
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd, nil
}

type CommandSink interface {
	Save(ctx context.Context, ref AggregateRef, cmd *PersistedCommand) error

	AppendBatch(ctx context.Context, ref AggregateRef, cmds []*PersistedCommand) error
}

type CommandSource interface {
	Load(ctx context.Context, ref AggregateRef) ([]*PersistedCommand, error)

	LoadFromTimestamp(
		ctx context.Context,
		ref AggregateRef,
		after time.Time,
	) ([]*PersistedCommand, error)

	LoadToTimestamp(
		ctx context.Context,
		ref AggregateRef,
		maxTime time.Time,
	) ([]*PersistedCommand, error)
}

type Store interface {
	CommandSink
	CommandSource
}

// CommandJournal reads all commands across all aggregates, ordered by
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
