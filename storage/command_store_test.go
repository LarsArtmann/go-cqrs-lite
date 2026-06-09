package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

func newTestCommandStore(t *testing.T) *storage.SQLCommandStore {
	t.Helper()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = storage.SQLiteInitSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("init schema: %v", err)
	}

	store, err := storage.NewSQLiteCommandStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCommandStore: %v", err)
	}

	return store
}

func testCommand(
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

func requireCommandCount(
	t *testing.T,
	commands []*command.PersistedCommand,
	want int,
	desc string,
) {
	t.Helper()

	if len(commands) != want {
		t.Fatalf("expected %d commands %s, got %d", want, desc, len(commands))
	}
}

func assertCommandID(t *testing.T, got, want id.CommandID, desc string) {
	t.Helper()
	if got != want {
		t.Errorf("%s command ID mismatch: got %v, want %v", desc, got, want)
	}
}

func TestSQLCommandStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	cmd1 := testCommand(t, "CreateUser", ref)
	cmd2 := testCommand(t, "UpdateUser", ref)

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

func TestSQLCommandStore_DuplicateCommand(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	cmd := testCommand(t, "CreateUser", ref)

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

func TestSQLCommandStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	cmd1 := testCommand(t, "CreateUser", ref)
	cmd2 := testCommand(t, "UpdateUser", ref)

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

func TestSQLCommandStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	_, err := store.Load(ctx, ref)
	if err == nil {
		t.Fatal("expected error for non-existent aggregate")
	}

	if !errors.Is(err, command.ErrCommandNotFound) {
		t.Errorf("expected ErrCommandNotFound, got: %v", err)
	}
}

func TestSQLCommandStore_LoadFromTimestamp(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

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

	if err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd2, cmd3}); err != nil {
		t.Fatalf("unexpected error on AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromTimestamp(ctx, ref, time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error on LoadFromTimestamp: %v", err)
	}

	requireCommandCount(t, loaded, 2, "after 2025-01-10")
	assertCommandID(t, loaded[0].ID(), cmd2.ID(), "first")
	assertCommandID(t, loaded[1].ID(), cmd3.ID(), "second")
}

func TestSQLCommandStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

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

	if err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{cmd1, cmd2}); err != nil {
		t.Fatalf("unexpected error on AppendBatch: %v", err)
	}

	loaded, err := store.LoadToTimestamp(ctx, ref, time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error on LoadToTimestamp: %v", err)
	}

	requireCommandCount(t, loaded, 2, "up to 2025-01-20")
	assertCommandID(t, loaded[0].ID(), cmd1.ID(), "first")
	assertCommandID(t, loaded[1].ID(), cmd2.ID(), "second")
}

func TestSQLCommandStore_Close(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	err := store.Close()
	if err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	cmd := testCommand(t, "CreateUser", ref)
	err = store.Save(ctx, ref, cmd)
	if err == nil {
		t.Fatal("expected error after Close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}
