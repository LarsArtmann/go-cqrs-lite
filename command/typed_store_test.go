package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
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

	ref := command.NewStreamRef("Todo", id.NewStreamID())

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

	ref := command.NewStreamRef("Todo", id.NewStreamID())

	md := command.Metadata{
		Custom: map[command.MetadataKey]string{"user_id": "user-123"},
	}

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

func TestTypedCommandStore_CBORPreservesActor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, codec.CBORCodec{})

	ref := command.NewStreamRef("Todo", id.NewStreamID())
	actor := id.NewUserActor(id.NewUserID())

	err := ts.Save(ctx, ref, command.TypedPersistedCommand[createTodoPayload]{
		Type:     "todo.create",
		Payload:  createTodoPayload{Title: "test"},
		Metadata: command.Metadata{Tracing: metadata.Tracing{ActorID: actor}},
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

	got := loaded[0].Metadata.ActorID
	if !got.Equal(actor) {
		t.Errorf("CBOR roundtrip lost actor: got %q, want %q",
			got.PrefixedString(), actor.PrefixedString())
	}
}

func TestTypedCommandStore_NilCodecDefaultsToCBOR(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, nil)

	ctx := context.Background()
	ref := command.NewStreamRef("Todo", id.NewStreamID())

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

func TestTypedCommandStore_LegacyPayloadsDecodeAcrossCodecs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := command.NewStreamRef("Todo", id.NewStreamID())

	seed := func(t *testing.T, data []byte) *memory.MemoryCommandStore {
		t.Helper()

		store := memory.NewMemoryCommandStore()
		pc, err := command.NewPersistedCommand("todo.create", ref, data)
		if err != nil {
			t.Fatalf("NewPersistedCommand: %v", err)
		}
		if err = store.Save(ctx, ref, pc); err != nil {
			t.Fatalf("seed Save: %v", err)
		}

		return store
	}

	rawJSON := []byte(`{"title":"legacy-json","description":"from disk"}`)

	rawCBOR, err := codec.CBORCodec{}.Encode(createTodoPayload{Title: "legacy-cbor"})
	if err != nil {
		t.Fatalf("encode raw CBOR: %v", err)
	}

	// Raw JSON payload read by the CBOR-default store.
	cborDefault := command.NewTypedCommandStore[createTodoPayload](seed(t, rawJSON), nil)
	loaded, err := cborDefault.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load legacy raw JSON: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Payload.Title != "legacy-json" {
		t.Fatalf("legacy raw JSON under CBOR default: got %+v", loaded)
	}

	// Raw CBOR payload read by a JSON-configured store.
	jsonConfigured := command.NewTypedCommandStore[createTodoPayload](
		seed(t, rawCBOR),
		codec.JSONCodec{},
	)
	loaded, err = jsonConfigured.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load legacy raw CBOR: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Payload.Title != "legacy-cbor" {
		t.Fatalf("legacy raw CBOR under JSON config: got %+v", loaded)
	}
}

func TestTypedCommandStore_AppendBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryCommandStore()

	ts := command.NewTypedCommandStore[createTodoPayload](store, codec.JSONCodec{})

	ref := command.NewStreamRef("Order", id.NewStreamID())

	cmds := []command.TypedPersistedCommand[createTodoPayload]{
		{Type: "order.create", Payload: createTodoPayload{Title: "first"}},
		{Type: "order.create", Payload: createTodoPayload{Title: "second"}},
		{Type: "order.create", Payload: createTodoPayload{Title: "third"}},
	}

	err := ts.AppendBatch(ctx, ref, cmds)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := ts.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(loaded))
	}

	if loaded[0].Payload.Title != "first" {
		t.Errorf("first payload Title = %q, want %q", loaded[0].Payload.Title, "first")
	}

	if loaded[2].Payload.Title != "third" {
		t.Errorf("third payload Title = %q, want %q", loaded[2].Payload.Title, "third")
	}
}
