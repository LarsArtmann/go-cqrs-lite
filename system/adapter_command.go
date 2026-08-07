package system

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// CommandAdapterOption tunes a CommandAdapter at construction time.
type CommandAdapterOption func(*CommandAdapter)

// WithCommandSerialization enables command serialization for persistent
// engines (SQLite, Pebble). When enabled, commands are encoded to JSON
// envelope strings on write and decoded on read. For the Memory engine,
// this option should NOT be set — commands are stored as direct pointers.
func WithCommandSerialization() CommandAdapterOption {
	return func(a *CommandAdapter) { a.serialize = true }
}

// CommandAdapter wraps a [metaengine.StreamLogBackend] as a [command.Store].
type CommandAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
	serialize  bool
}

// NewCommandAdapter creates a command.Store backed by a StreamLogBackend.
func NewCommandAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...CommandAdapterOption,
) *CommandAdapter {
	a := &CommandAdapter{backend: backend, collection: collection}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

var (
	_ command.Store                  = (*CommandAdapter)(nil)
	_ command.SeekableCommandJournal = (*CommandAdapter)(nil)
)

func (a *CommandAdapter) Save(
	ctx context.Context,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	return a.backend.StreamAppend(ctx, a.collection, ref.StreamKey(), a.commandsToAny([]*command.PersistedCommand{cmd}))
}

func (a *CommandAdapter) AppendBatch(
	ctx context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	return a.backend.StreamAppend(ctx, a.collection, ref.StreamKey(), a.commandsToAny(cmds))
}

func (a *CommandAdapter) Load(
	ctx context.Context, ref command.StreamRef,
) ([]*command.PersistedCommand, error) {
	values, err := a.backend.StreamRead(ctx, a.collection, ref.StreamKey())
	if err != nil {
		return nil, fmt.Errorf("command adapter: load: %w", err)
	}

	return a.anyToCommands(values)
}

func (a *CommandAdapter) LoadFromTimestamp(
	ctx context.Context, ref command.StreamRef, after time.Time,
) ([]*command.PersistedCommand, error) {
	return a.loadFiltered(ctx, ref, func(cmd *command.PersistedCommand) bool {
		return cmd.ReceivedAt().After(after)
	})
}

func (a *CommandAdapter) LoadToTimestamp(
	ctx context.Context, ref command.StreamRef, maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return a.loadFiltered(ctx, ref, func(cmd *command.PersistedCommand) bool {
		return !cmd.ReceivedAt().After(maxTime)
	})
}

func (a *CommandAdapter) loadFiltered(
	ctx context.Context, ref command.StreamRef,
	keep func(cmd *command.PersistedCommand) bool,
) ([]*command.PersistedCommand, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	var result []*command.PersistedCommand

	for _, cmd := range all {
		if keep(cmd) {
			result = append(result, cmd)
		}
	}

	return result, nil
}

func (a *CommandAdapter) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("command adapter: read all: %w", err)
	}

	return a.anyToCommands(values)
}

func (a *CommandAdapter) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	afterSeq := int64(0)

	if afterCommandID != (id.CommandID{}) {
		all, err := a.backend.JournalReadAll(ctx, a.collection)
		if err != nil {
			return nil, fmt.Errorf("command adapter: read from: %w", err)
		}

		cmds, err := a.anyToCommands(all)
		if err != nil {
			return nil, fmt.Errorf("command adapter: read from: %w", err)
		}

		for i, cmd := range cmds {
			if cmd.ID() == afterCommandID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("command adapter: read from: %w", err)
	}

	return a.anyToCommands(values)
}
