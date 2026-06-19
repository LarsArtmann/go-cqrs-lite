package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

type createTodoPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func TestTypedCommandStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, codec.JSONCodec{})

	ref := command.NewAggregateRef("Todo", id.NewAggregateID())

	err := ts.Save(ctx, ref, command.TypedPersistedCommand[createTodoPayload]{
		Type:    "todo.create",
		Payload: createTodoPayload{Title: "buy milk", Description: "from the store"},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := ts.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	if loaded[0].Payload.Title != "buy milk" {
		t.Errorf("Title = %q, want %q", loaded[0].Payload.Title, "buy milk")
	}

	if loaded[0].Payload.Description != "from the store" {
		t.Errorf("Description = %q, want %q", loaded[0].Payload.Description, "from the store")
	}

	if loaded[0].Type != "todo.create" {
		t.Errorf("Type = %q, want %q", loaded[0].Type, "todo.create")
	}
}

func TestTypedCommandStore_PreservesMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, codec.JSONCodec{})

	ref := command.NewAggregateRef("Todo", id.NewAggregateID())

	md := command.NewMetadata()
	event.EnsureCustom(&md)
	md.Custom["user_id"] = "user-123"

	err := ts.Save(ctx, ref, command.TypedPersistedCommand[createTodoPayload]{
		Type:     "todo.create",
		Payload:  createTodoPayload{Title: "test"},
		Metadata: md,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := ts.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	got := loaded[0].Metadata.Custom["user_id"]
	if got != "user-123" {
		t.Errorf("metadata user_id = %q, want %q", got, "user-123")
	}
}

func TestTypedCommandStore_NilCodecDefaultsToJSON(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, nil)

	ctx := context.Background()
	ref := command.NewAggregateRef("Todo", id.NewAggregateID())

	err := ts.Save(ctx, ref, command.TypedPersistedCommand[createTodoPayload]{
		Type:    "todo.create",
		Payload: createTodoPayload{Title: "nil codec test"},
	})
	if err != nil {
		t.Fatalf("Save with nil codec: %v", err)
	}

	loaded, err := ts.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded[0].Payload.Title != "nil codec test" {
		t.Errorf("Title = %q, want %q", loaded[0].Payload.Title, "nil codec test")
	}
}
