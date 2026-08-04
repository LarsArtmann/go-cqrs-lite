package system

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// CommandAdapter wraps a [metaengine.StreamLogBackend] as a [command.Store].
// Commands are stream-keyed append-only logs, just like events. This adapter
// provides command audit-trail persistence without duplicating storage logic.
type CommandAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
}

// NewCommandAdapter creates a command.Store backed by a StreamLogBackend.
func NewCommandAdapter(backend metaengine.StreamLogBackend, collection string) *CommandAdapter {
	return &CommandAdapter{backend: backend, collection: collection}
}

// Compile-time assertion.
var _ command.Store = (*CommandAdapter)(nil)

func (a *CommandAdapter) Save(ctx context.Context, ref id.StreamRef, cmd command.PersistedCommand) error {
	sid := ref.StreamKey()

	return a.backend.StreamAppend(ctx, a.collection, sid, []any{cmd})
}

func (a *CommandAdapter) Load(ctx context.Context, ref id.StreamRef) ([]command.PersistedCommand, error) {
	sid := ref.StreamKey()

	values, err := a.backend.StreamRead(ctx, a.collection, sid)
	if err != nil {
		return nil, fmt.Errorf("command adapter: load: %w", err)
	}

	result := make([]command.PersistedCommand, 0, len(values))
	for _, val := range values {
		cmd, ok := val.(command.PersistedCommand)
		if !ok {
			return nil, fmt.Errorf("command adapter: value is not PersistedCommand (got %T)", val)
		}

		result = append(result, cmd)
	}

	return result, nil
}

func (a *CommandAdapter) ReadAll(ctx context.Context) ([]command.PersistedCommand, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("command adapter: read all: %w", err)
	}

	result := make([]command.PersistedCommand, 0, len(values))
	for _, val := range values {
		cmd, ok := val.(command.PersistedCommand)
		if !ok {
			return nil, fmt.Errorf("command adapter: value is not PersistedCommand (got %T)", val)
		}

		result = append(result, cmd)
	}

	return result, nil
}

func (a *CommandAdapter) ReadFrom(
	ctx context.Context,
	afterCommandID id.EventID,
	limit int,
) ([]command.PersistedCommand, error) {
	afterSeq := int64(0)
	if afterCommandID != "" {
		all, err := a.backend.JournalReadAll(ctx, a.collection)
		if err != nil {
			return nil, fmt.Errorf("command adapter: read from: %w", err)
		}

		for i, val := range all {
			cmd, ok := val.(command.PersistedCommand)
			if ok && cmd.ID == afterCommandID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("command adapter: read from: %w", err)
	}

	result := make([]command.PersistedCommand, 0, len(values))
	for _, val := range values {
		cmd, ok := val.(command.PersistedCommand)
		if !ok {
			return nil, fmt.Errorf("command adapter: value is not PersistedCommand (got %T)", val)
		}

		result = append(result, cmd)
	}

	return result, nil
}
