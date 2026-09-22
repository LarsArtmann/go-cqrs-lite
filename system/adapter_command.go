package system

import (
	"context"
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
	return func(a *CommandAdapter) { a.Serialize = true }
}

// CommandAdapter wraps a [metaengine.StreamLogBackend] as a [command.Store].
type CommandAdapter struct {
	AdapterCore[*command.PersistedCommand]
}

// NewCommandAdapter creates a command.Store backed by a StreamLogBackend.
func NewCommandAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...CommandAdapterOption,
) *CommandAdapter {
	a := &CommandAdapter{}
	a.AdapterCore = AdapterCore[*command.PersistedCommand]{
		Backend:    backend,
		Collection: collection,
		Noun:       "command",
		Encode:     a.encodeCommand,
		Decode:     a.decodeCommand,
		IDOf:       func(cmd *command.PersistedCommand) string { return cmd.ID().String() },
	}

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
	//art-dupl:accept same-shape ToAny+wrap — adapter-specific types
	values, err := a.ToAny([]*command.PersistedCommand{cmd})
	if err != nil {
		return err
	}

	return a.Backend.StreamAppend(ctx, a.Collection, ref.StreamKey(), values)
}

func (a *CommandAdapter) AppendBatch(
	ctx context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	//art-dupl:accept same-shape ToAny+wrap — adapter-specific types
	values, err := a.ToAny(cmds)
	if err != nil {
		return err
	}

	return a.Backend.StreamAppend(ctx, a.Collection, ref.StreamKey(), values)
}

func (a *CommandAdapter) Load(
	ctx context.Context, ref command.StreamRef,
) ([]*command.PersistedCommand, error) {
	return a.LoadStream(ctx, ref.StreamKey())
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

// ReadAll is promoted from AdapterCore and satisfies command.CommandJournal.

func (a *CommandAdapter) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	after := ""
	if afterCommandID != (id.CommandID{}) {
		after = afterCommandID.String()
	}

	return a.ReadFromAfter(ctx, after, limit)
}
