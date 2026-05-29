package command

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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
func (c *PersistedCommand) Metadata() Metadata { return c.metadata }

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
	return func(c *PersistedCommand) { c.metadata = m }
}

func NewPersistedCommand(
	cmdType Type,
	ref AggregateRef,
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
			ErrEmptyAggregateType,
			"command.empty_aggregate_type",
			"aggregate type is required in ref",
		)
	}

	if ref.ID.IsZero() {
		return nil, errorfamily.WrapRejection(
			ErrNilAggregateID,
			"command.nil_aggregate_id",
			"aggregate ID is required in ref",
		)
	}

	var payloadCopy []byte
	if payload != nil {
		payloadCopy = make([]byte, len(payload))
		copy(payloadCopy, payload)
	}

	cmd := &PersistedCommand{
		id:           id.NewCommandID(),
		cmdType:      cmdType,
		aggregateRef: ref,
		receivedAt:   time.Now(),
		payload:      payloadCopy,
		metadata:     Metadata{},
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd, nil
}

type CommandSink interface {
	io.Closer

	Save(ctx context.Context, ref AggregateRef, cmd *PersistedCommand) error

	AppendBatch(ctx context.Context, ref AggregateRef, cmds []*PersistedCommand) error
}

type CommandSource interface {
	io.Closer

	Load(ctx context.Context, ref AggregateRef) ([]*PersistedCommand, error)

	LoadFromTimestamp(ctx context.Context, ref AggregateRef, after time.Time) ([]*PersistedCommand, error)

	LoadToTimestamp(ctx context.Context, ref AggregateRef, maxTime time.Time) ([]*PersistedCommand, error)
}

type Store interface {
	CommandSink
	CommandSource
}
