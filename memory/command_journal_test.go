package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func TestMemoryCommandStore_Journal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	cmds := make([]*command.PersistedCommand, 3)
	for i := range cmds {
		cmd, err := command.NewPersistedCommand(
			"user.create", ref, []byte(`{}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}

	err := store.AppendBatch(ctx, ref, cmds)
	if err != nil {
		t.Fatal(err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}

	result, err := store.ReadFrom(ctx, cmds[0].ID(), 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 commands after first, got %d", len(result))
	}

	limited, err := store.ReadFrom(ctx, cmds[0].ID(), 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(limited) != 1 {
		t.Fatalf("expected 1 command with limit=1, got %d", len(limited))
	}
}

func TestMemoryCommandStore_Journal_ReadFromZeroID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	cmds := make([]*command.PersistedCommand, 3)
	for i := range cmds {
		cmd, err := command.NewPersistedCommand(
			"user.create", ref, []byte(`{}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}

	if err := store.AppendBatch(ctx, ref, cmds); err != nil {
		t.Fatal(err)
	}

	result, err := store.ReadFrom(ctx, id.CommandID{}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 commands from start, got %d", len(result))
	}
}

func TestMemoryCommandStore_Journal_EmptyStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()
	defer func() { _ = store.Close() }()

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 0 {
		t.Fatalf("expected 0 commands from empty store, got %d", len(all))
	}

	result, err := store.ReadFrom(ctx, id.CommandID{}, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 commands from empty store ReadFrom, got %d", len(result))
	}
}

func TestMemoryCommandStore_Journal_ReadFromNonExistentID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	cmd, err := command.NewPersistedCommand("user.create", ref, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatal(err)
	}

	otherID := id.NewCommandID()

	result, err := store.ReadFrom(ctx, otherID, 10)
	if err != nil {
		t.Fatal(err)
	}

	if result != nil {
		t.Fatalf("expected nil for non-existent command ID, got %d results", len(result))
	}
}

func TestMemoryCommandStore_Journal_ClosedStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := store.ReadAll(ctx)
	if err == nil {
		t.Fatal("expected error on ReadAll after close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}

	_, err = store.ReadFrom(ctx, id.CommandID{}, 10)
	if err == nil {
		t.Fatal("expected error on ReadFrom after close")
	}

	if !errors.Is(err, command.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}

func TestMemoryCommandStore_Journal_OrderingByReceivedAt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	cmds := make([]*command.PersistedCommand, 3)
	for i, ts := range []time.Time{t1, t2, t3} {
		cmd, err := command.NewPersistedCommand(
			"user.update", ref, []byte(`{}`),
			command.WithReceivedAt(ts),
		)
		if err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}

	for _, cmd := range cmds {
		if err := store.Save(ctx, ref, cmd); err != nil {
			t.Fatal(err)
		}
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}

	for i, expected := range cmds {
		if all[i].ID() != expected.ID() {
			t.Errorf("position %d: expected %s, got %s", i, expected.ID(), all[i].ID())
		}
	}
}
