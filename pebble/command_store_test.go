package pebble_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v2"
)

func newCommandStore(t *testing.T) *cqrspebble.CommandStore {
	t.Helper()

	dir := t.TempDir()

	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	return cqrspebble.NewCommandStore(database, slog.Default())
}

func mustCreateCommand(
	t *testing.T,
	cmdType string,
	ref command.AggregateRef,
) *command.PersistedCommand {
	t.Helper()

	cmd, err := command.NewPersistedCommand(
		command.Type(cmdType),
		ref,
		[]byte(`{"action":"test"}`),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	return cmd
}

func TestCommandStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	cmd := mustCreateCommand(t, "user.create", ref)

	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	if loaded[0].ID() != cmd.ID() {
		t.Errorf("ID mismatch: got %s, want %s", loaded[0].ID(), cmd.ID())
	}

	if loaded[0].Type() != cmd.Type() {
		t.Errorf("Type mismatch: got %s, want %s", loaded[0].Type(), cmd.Type())
	}
}

func TestCommandStore_DuplicateDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	cmd := mustCreateCommand(t, "user.create", ref)

	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := store.Save(ctx, ref, cmd)
	if !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatalf("expected ErrDuplicateCommand, got %v", err)
	}
}

func TestCommandStore_AppendBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("Order", aggID)

	cmds := []*command.PersistedCommand{
		mustCreateCommand(t, "order.create", ref),
		mustCreateCommand(t, "order.confirm", ref),
		mustCreateCommand(t, "order.ship", ref),
	}

	if err := store.AppendBatch(ctx, ref, cmds); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(loaded))
	}
}

func TestCommandStore_ReadAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	ref1 := command.NewAggregateRef("User", id.NewAggregateID())
	ref2 := command.NewAggregateRef("Order", id.NewAggregateID())

	cmd1 := mustCreateCommand(t, "user.create", ref1)
	if err := store.Save(ctx, ref1, cmd1); err != nil {
		t.Fatalf("Save cmd1: %v", err)
	}

	time.Sleep(2 * time.Millisecond) // ensure ULID time ordering

	cmd2 := mustCreateCommand(t, "order.create", ref2)
	if err := store.Save(ctx, ref2, cmd2); err != nil {
		t.Fatalf("Save cmd2: %v", err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 commands in journal, got %d", len(all))
	}
}

func TestCommandStore_ReadFrom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	ref := command.NewAggregateRef("User", id.NewAggregateID())

	cmdIDs := make([]id.CommandID, 0, 5)

	for i := range 5 {
		cmd := mustCreateCommand(t, "user.action", ref)
		cmdIDs = append(cmdIDs, cmd.ID())

		if err := store.Save(ctx, ref, cmd); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}

		time.Sleep(2 * time.Millisecond) // ensure ULID ordering
	}

	// Read from beginning
	all, err := store.ReadFrom(ctx, id.CommandID{}, 0)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}

	// Read after 3rd command with limit 2
	page, err := store.ReadFrom(ctx, cmdIDs[2], 2)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(page) != 2 {
		t.Fatalf("expected 2 in page, got %d", len(page))
	}

	if page[0].ID() != cmdIDs[3] {
		t.Errorf("expected first in page to be cmd 4, got %s", page[0].ID())
	}
}

func TestCommandStore_LoadFromTimestamp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newCommandStore(t)

	ref := command.NewAggregateRef("User", id.NewAggregateID())

	before := time.Now()

	cmd1 := mustCreateCommand(t, "user.create", ref)
	if err := store.Save(ctx, ref, cmd1); err != nil {
		t.Fatalf("Save cmd1: %v", err)
	}

	midpoint := time.Now()
	time.Sleep(2 * time.Millisecond)

	cmd2 := mustCreateCommand(t, "user.update", ref)
	if err := store.Save(ctx, ref, cmd2); err != nil {
		t.Fatalf("Save cmd2: %v", err)
	}

	// Load all after midpoint → should get only cmd2
	filtered, err := store.LoadFromTimestamp(ctx, ref, midpoint)
	if err != nil {
		t.Fatalf("LoadFromTimestamp: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 command after midpoint, got %d", len(filtered))
	}

	// Load all after before → should get both
	all, err := store.LoadFromTimestamp(ctx, ref, before)
	if err != nil {
		t.Fatalf("LoadFromTimestamp all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 commands after before, got %d", len(all))
	}
}
