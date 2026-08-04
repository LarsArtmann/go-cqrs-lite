package system

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// CommandAdapter wraps a [metaengine.StreamLogBackend] as a [command.Store].
type CommandAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
}

// NewCommandAdapter creates a command.Store backed by a StreamLogBackend.
func NewCommandAdapter(backend metaengine.StreamLogBackend, collection string) *CommandAdapter {
	return &CommandAdapter{backend: backend, collection: collection}
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
	return a.backend.StreamAppend(ctx, a.collection, ref.StreamKey(), []any{cmd})
}

func (a *CommandAdapter) AppendBatch(
	ctx context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	values := make([]any, len(cmds))
	for i, c := range cmds {
		values[i] = c
	}

	return a.backend.StreamAppend(ctx, a.collection, ref.StreamKey(), values)
}

func (a *CommandAdapter) Load(
	ctx context.Context, ref command.StreamRef,
) ([]*command.PersistedCommand, error) {
	values, err := a.backend.StreamRead(ctx, a.collection, ref.StreamKey())
	if err != nil {
		return nil, fmt.Errorf("command adapter: load: %w", err)
	}

	return anyToCommands(values), nil
}

func (a *CommandAdapter) LoadFromTimestamp(
	ctx context.Context, ref command.StreamRef, after time.Time,
) ([]*command.PersistedCommand, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	var result []*command.PersistedCommand

	for _, cmd := range all {
		if cmd.ReceivedAt().After(after) {
			result = append(result, cmd)
		}
	}

	return result, nil
}

func (a *CommandAdapter) LoadToTimestamp(
	ctx context.Context, ref command.StreamRef, maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	var result []*command.PersistedCommand

	for _, cmd := range all {
		if !cmd.ReceivedAt().After(maxTime) {
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

	return anyToCommands(values), nil
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

		for i, val := range all {
			cmd, ok := val.(*command.PersistedCommand)
			if ok && cmd.ID() == afterCommandID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("command adapter: read from: %w", err)
	}

	return anyToCommands(values), nil
}

func anyToCommands(values []any) []*command.PersistedCommand {
	result := make([]*command.PersistedCommand, 0, len(values))
	for _, val := range values {
		cmd, ok := val.(*command.PersistedCommand)
		if !ok {
			continue
		}

		result = append(result, cmd)
	}

	return result
}
