package bbolt

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4/commandtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestCommandStoreSuite(t *testing.T) {
	t.Parallel()

	commandtest.RunStoreSuite(t, func(t *testing.T) commandtest.StoreSuite {
		return newTestBackend(t).CommandStore()
	})
}

func TestCommandStore_AppendBatchDuplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backend := newTestBackend(t)
	store := backend.CommandStore()

	streamID := id.NewStreamID()
	ref := command.NewStreamRef("User", streamID)

	dup := commandtest.MustCreateCommand(t, "user.create", ref)

	cmds := []*command.PersistedCommand{
		commandtest.MustCreateCommand(t, "user.update", ref),
		dup,
	}

	if err := store.AppendBatch(ctx, ref, cmds); err != nil {
		t.Fatalf("first AppendBatch: %v", err)
	}

	err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{dup})
	if !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatalf("expected ErrDuplicateCommand on batch with dup, got %v", err)
	}
}

func TestCommandStore_LoadEmptyStream(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backend := newTestBackend(t)
	store := backend.CommandStore()

	ref := command.NewStreamRef("User", id.NewStreamID())

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load empty stream: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected 0 commands for empty stream, got %d", len(loaded))
	}
}
