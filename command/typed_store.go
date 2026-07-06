package command

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// TypedPersistedCommand is a persisted command with a typed payload P, closing
// the type-safety hole where [PersistedCommand].Payload is an untyped []byte.
//
// Every consumer that loads a persisted command today must manually decode
// Payload, with no compile-time guarantee that the bytes match the expected
// type. TypedPersistedCommand makes that decode the adapter's responsibility:
// a [TypedCommandStore] decodes once at the boundary.
type TypedPersistedCommand[P any] struct {
	ID           id.CommandID
	Type         Type
	AggregateRef AggregateRef
	ReceivedAt   time.Time
	Payload      P
	Metadata     Metadata
}

// TypedCommandStore adapts an untyped [Store] plus a [codec.Codec] into a
// typed interface over P. It handles encode/decode at the store boundary.
//
//	ts := command.NewTypedCommandStore[CreateTodoPayload](store, codec.JSONCodec{})
//	_ = ts.Save(ctx, ref, command.TypedPersistedCommand[CreateTodoPayload]{...})
//	loaded, _ := ts.Load(ctx, ref)
//	// loaded[0].Payload is CreateTodoPayload, not []byte
type TypedCommandStore[P any] struct {
	store Store
	codec codec.Codec
}

// NewTypedCommandStore creates a typed adapter over store using c for payload
// serialization. If c is nil, [codec.JSONCodec] is used.
func NewTypedCommandStore[P any](store Store, c codec.Codec) *TypedCommandStore[P] {
	if c == nil {
		c = codec.JSONCodec{}
	}

	return &TypedCommandStore[P]{store: store, codec: c}
}

// Save encodes cmd.Payload and delegates to the underlying [Store].
func (t *TypedCommandStore[P]) Save(
	ctx context.Context,
	ref AggregateRef,
	cmd TypedPersistedCommand[P],
) error {
	data, err := t.codec.Encode(cmd.Payload)
	if err != nil {
		return errorfamily.WrapCorruption(err, "command.typed_store.encode",
			"encode typed payload")
	}

	opts := []PersistOption{
		WithCommandMetadata(cmd.Metadata),
	}

	if cmd.ReceivedAt.IsZero() {
		opts = append(opts, WithReceivedAt(time.Now()))
	} else {
		opts = append(opts, WithReceivedAt(cmd.ReceivedAt))
	}

	if cmd.ID != (id.CommandID{}) {
		opts = append(opts, WithPersistedCommandID(cmd.ID))
	}

	persisted, err := NewPersistedCommand(cmd.Type, ref, data, opts...)
	if err != nil {
		return err
	}

	err = t.store.Save(ctx, ref, persisted)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "command.typed_store.save", "save typed command")
	}

	return nil
}

// AppendBatch persists multiple typed commands atomically. Each command's
// payload is encoded via the codec before delegating to the underlying [Store].
func (t *TypedCommandStore[P]) AppendBatch(
	ctx context.Context,
	ref AggregateRef,
	cmds []TypedPersistedCommand[P],
) error {
	persisted := make([]*PersistedCommand, 0, len(cmds))

	for i, cmd := range cmds {
		data, err := t.codec.Encode(cmd.Payload)
		if err != nil {
			return errorfamily.WrapCorruption(err, "command.typed_store.encode_batch",
				fmt.Sprintf("encode typed payload at index %d", i))
		}

		opts := []PersistOption{
			WithCommandMetadata(cmd.Metadata),
		}

		if cmd.ReceivedAt.IsZero() {
			opts = append(opts, WithReceivedAt(time.Now()))
		} else {
			opts = append(opts, WithReceivedAt(cmd.ReceivedAt))
		}

		if cmd.ID != (id.CommandID{}) {
			opts = append(opts, WithPersistedCommandID(cmd.ID))
		}

		p, err := NewPersistedCommand(cmd.Type, ref, data, opts...)
		if err != nil {
			return err
		}

		persisted = append(persisted, p)
	}

	err := t.store.AppendBatch(ctx, ref, persisted)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"command.typed_store.append_batch",
			"append typed commands",
		)
	}

	return nil
}

// Load retrieves all commands for ref, decoding each payload into P.
func (t *TypedCommandStore[P]) Load(
	ctx context.Context,
	ref AggregateRef,
) ([]TypedPersistedCommand[P], error) {
	cmds, err := t.store.Load(ctx, ref)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"command.typed_store.load",
			"load typed commands",
		)
	}

	result := make([]TypedPersistedCommand[P], 0, len(cmds))

	for _, cmd := range cmds {
		var payload P

		err := t.codec.Decode(cmd.Payload(), &payload)
		if err != nil {
			return nil, errorfamily.WrapCorruption(err, "command.typed_store.decode",
				fmt.Sprintf("decode typed payload for %s", cmd.ID()))
		}

		result = append(result, TypedPersistedCommand[P]{
			ID:           cmd.ID(),
			Type:         cmd.Type(),
			AggregateRef: cmd.AggregateRef(),
			ReceivedAt:   cmd.ReceivedAt(),
			Payload:      payload,
			Metadata:     cmd.Metadata(),
		})
	}

	return result, nil
}
