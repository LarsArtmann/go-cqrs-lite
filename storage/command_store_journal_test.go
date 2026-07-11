package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func TestSQLCommandStore_ReadAll(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	cmd1 := testCommand(t, "CreateUser", ref)
	cmd2 := testCommand(t, "UpdateUser", ref)

	if err := store.Save(ctx, ref, cmd1); err != nil {
		t.Fatalf("Save cmd1: %v", err)
	}

	if err := store.Save(ctx, ref, cmd2); err != nil {
		t.Fatalf("Save cmd2: %v", err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	requireCommandCount(t, all, 2, "ReadAll")
	assertCommandID(t, all[0].ID(), cmd1.ID(), "ReadAll first")
	assertCommandID(t, all[1].ID(), cmd2.ID(), "ReadAll second")
}

func TestSQLCommandStore_ReadAll_Empty(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll empty: %v", err)
	}

	requireCommandCount(t, all, 0, "empty ReadAll")
}

func TestSQLCommandStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	cmds := make([]*command.PersistedCommand, 3)
	for i := range cmds {
		cmds[i] = testCommand(t, command.Type("cmd."+string(rune('A'+i))), ref)
	}

	for _, cmd := range cmds {
		if err := store.Save(ctx, ref, cmd); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	result, err := store.ReadFrom(ctx, id.CommandID{}, 2)
	if err != nil {
		t.Fatalf("ReadFrom zero ID: %v", err)
	}

	requireCommandCount(t, result, 2, "ReadFrom zero limit=2")
}

func TestSQLCommandStore_ReadFrom_AfterID(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	cmd1, _ := command.NewPersistedCommand("Create", ref, nil, command.WithReceivedAt(t1))
	cmd2, _ := command.NewPersistedCommand("Update", ref, nil, command.WithReceivedAt(t2))
	cmd3, _ := command.NewPersistedCommand("Delete", ref, nil, command.WithReceivedAt(t3))

	for _, cmd := range []*command.PersistedCommand{cmd1, cmd2, cmd3} {
		if err := store.Save(ctx, ref, cmd); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	result, err := store.ReadFrom(ctx, cmd1.ID(), 10)
	if err != nil {
		t.Fatalf("ReadAfter cmd1: %v", err)
	}

	requireCommandCount(t, result, 2, "after cmd1")
	assertCommandID(t, result[0].ID(), cmd2.ID(), "after cmd1 first")
	assertCommandID(t, result[1].ID(), cmd3.ID(), "after cmd1 second")
}

func TestSQLCommandStore_ReadAll_Closed(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := store.ReadAll(ctx)
	if err == nil {
		t.Fatal("expected error on ReadAll after close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}

func TestSQLCommandStore_ReadFrom_Closed(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := store.ReadFrom(ctx, id.CommandID{}, 10)
	if err == nil {
		t.Fatal("expected error on ReadFrom after close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}

func TestSQLCommandStore_ReadFrom_NonExistentID(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	cmd := testCommand(t, "CreateUser", ref)
	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	otherID := id.NewCommandID()
	result, err := store.ReadFrom(ctx, otherID, 10)
	if err != nil {
		t.Fatalf("ReadFrom non-existent: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 results for non-existent ID, got %d", len(result))
	}
}

func TestSQLCommandStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store := newTestCommandStore(t)
	ctx := context.Background()
	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	meta := command.NewMetadata()
	meta.CorrelationID = id.NewCorrelationID()
	meta.UserID = id.NewUserID()
	command.EnsureCustom(&meta)
	meta.Custom["source"] = "test"

	cmd, err := command.NewPersistedCommand(
		"CreateUser", ref, []byte(`{"name":"Alice"}`),
		command.WithCommandMetadata(meta),
	)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

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

	got := loaded[0].Metadata()
	if got.CorrelationID != meta.CorrelationID {
		t.Errorf("CorrelationID mismatch: got %v, want %v", got.CorrelationID, meta.CorrelationID)
	}

	if got.UserID != meta.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", got.UserID, meta.UserID)
	}

	if got.Custom["source"] != "test" {
		t.Errorf("Custom[source] = %q, want %q", got.Custom["source"], "test")
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 command in journal, got %d", len(all))
	}

	gotJournal := all[0].Metadata()
	if gotJournal.CorrelationID != meta.CorrelationID {
		t.Errorf(
			"Journal CorrelationID mismatch: got %v, want %v",
			gotJournal.CorrelationID,
			meta.CorrelationID,
		)
	}
}
