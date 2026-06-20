package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func validCommandRef() command.AggregateRef {
	return command.NewAggregateRef("User", parseAggID("01HK1540X0841Y0A6BSX1VKR95"))
}

func testPersistedCommand(
	t *testing.T,
	cmdType command.Type,
	ref command.AggregateRef,
) *command.PersistedCommand {
	t.Helper()

	cmd, err := command.NewPersistedCommand(cmdType, ref, []byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("create test command: %v", err)
	}

	return cmd
}

func TestMemoryCommandStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	cmd1 := testPersistedCommand(t, "CreateUser", ref)
	cmd2 := testPersistedCommand(t, "UpdateUser", ref)

	err := store.Save(ctx, ref, cmd1)
	if err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	err = store.Save(ctx, ref, cmd2)
	if err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(loaded))
	}

	if loaded[0].ID() != cmd1.ID() {
		t.Errorf("first command ID mismatch: got %v, want %v", loaded[0].ID(), cmd1.ID())
	}

	if loaded[1].ID() != cmd2.ID() {
		t.Errorf("second command ID mismatch: got %v, want %v", loaded[1].ID(), cmd2.ID())
	}
}

func TestMemoryCommandStore_DuplicateCommand(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	cmd := testPersistedCommand(t, "CreateUser", ref)

	err := store.Save(ctx, ref, cmd)
	if err != nil {
		t.Fatalf("unexpected error on first Save: %v", err)
	}

	err = store.Save(ctx, ref, cmd)
	if err == nil {
		t.Fatal("expected error for duplicate command")
	}

	if !errors.Is(err, command.ErrDuplicateCommand) {
		t.Errorf("expected ErrDuplicateCommand, got: %v", err)
	}
}

func TestMemoryCommandStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	cmd1 := testPersistedCommand(t, "CreateUser", ref)
	cmd2 := testPersistedCommand(t, "UpdateUser", ref)

	err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd1, cmd2})
	if err != nil {
		t.Fatalf("unexpected error on AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(loaded))
	}
}

func TestMemoryCommandStore_AppendBatch_DuplicateInBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	cmd := testPersistedCommand(t, "CreateUser", ref)

	err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd, cmd})
	if err == nil {
		t.Fatal("expected error for duplicate command in batch")
	}

	if !errors.Is(err, command.ErrDuplicateCommand) {
		t.Errorf("expected ErrDuplicateCommand, got: %v", err)
	}
}

func TestMemoryCommandStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	_, err := store.Load(ctx, ref)
	if err == nil {
		t.Fatal("expected error for non-existent aggregate")
	}

	if !errors.Is(err, command.ErrCommandNotFound) {
		t.Errorf("expected ErrCommandNotFound, got: %v", err)
	}
}

func setupTimestampCommands(
	t *testing.T,
) (*command.PersistedCommand, *command.PersistedCommand, *command.PersistedCommand, command.AggregateRef) {
	t.Helper()
	ref := validCommandRef()

	cmd1, err := command.NewPersistedCommand("CreateUser", ref, nil, command.WithReceivedAt(
		time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
	))
	if err != nil {
		t.Fatalf("create test command: %v", err)
	}

	cmd2, err := command.NewPersistedCommand("UpdateUser", ref, nil, command.WithReceivedAt(
		time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	))
	if err != nil {
		t.Fatalf("create test command: %v", err)
	}

	cmd3, err := command.NewPersistedCommand("DeleteUser", ref, nil, command.WithReceivedAt(
		time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC),
	))
	if err != nil {
		t.Fatalf("create test command: %v", err)
	}

	return cmd1, cmd2, cmd3, ref
}

func assertCommandCount(t *testing.T, loaded []*command.PersistedCommand, want int, desc string) {
	t.Helper()
	if len(loaded) != want {
		t.Fatalf("expected %d commands %s, got %d", want, desc, len(loaded))
	}
}

func checkCommandID(t *testing.T, got, want id.CommandID, desc string) {
	t.Helper()
	if got != want {
		t.Errorf("%s ID mismatch: got %v, want %v", desc, got, want)
	}
}

func TestMemoryCommandStore_LoadFromTimestamp(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	_, cmd2, cmd3, ref := setupTimestampCommands(t)

	if err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd2, cmd3}); err != nil {
		t.Fatalf("unexpected error on AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromTimestamp(ctx, ref, time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error on LoadFromTimestamp: %v", err)
	}

	assertCommandCount(t, loaded, 2, "after 2025-01-10")
	checkCommandID(t, loaded[0].ID(), cmd2.ID(), "first")
	checkCommandID(t, loaded[1].ID(), cmd3.ID(), "second")
}

func TestMemoryCommandStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	cmd1, cmd2, _, ref := setupTimestampCommands(t)

	if err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd1, cmd2}); err != nil {
		t.Fatalf("unexpected error on AppendBatch: %v", err)
	}

	loaded, err := store.LoadToTimestamp(ctx, ref, time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error on LoadToTimestamp: %v", err)
	}

	assertCommandCount(t, loaded, 2, "up to 2025-01-20")
	checkCommandID(t, loaded[0].ID(), cmd1.ID(), "first")
	checkCommandID(t, loaded[1].ID(), cmd2.ID(), "second")
}

func TestMemoryCommandStore_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()
	ref := validCommandRef()

	err := store.Close()
	if err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	cmd := testPersistedCommand(t, "CreateUser", ref)
	err = store.Save(ctx, ref, cmd)
	if err == nil {
		t.Fatal("expected error after Close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}

func TestMemoryCommandStore_MultipleAggregates(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()
	ctx := context.Background()

	ref1 := command.NewAggregateRef("User", parseAggID("01HK1540X0841Y0A6BSX1VKR95"))
	ref2 := command.NewAggregateRef("Order", parseAggID("01HK1540X0841Y0A6BSX1VKR96"))

	cmd1 := testPersistedCommand(t, "CreateUser", ref1)
	cmd2 := testPersistedCommand(t, "CreateOrder", ref2)

	err := store.Save(ctx, ref1, cmd1)
	if err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	err = store.Save(ctx, ref2, cmd2)
	if err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	loaded1, err := store.Load(ctx, ref1)
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}

	if len(loaded1) != 1 {
		t.Fatalf("expected 1 command for User, got %d", len(loaded1))
	}

	loaded2, err := store.Load(ctx, ref2)
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}

	if len(loaded2) != 1 {
		t.Fatalf("expected 1 command for Order, got %d", len(loaded2))
	}
}
