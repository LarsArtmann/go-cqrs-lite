package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func TestMemoryCommandStore_Journal(t *testing.T) {
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
