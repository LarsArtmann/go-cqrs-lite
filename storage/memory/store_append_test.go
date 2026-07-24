package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	streamID := idtest.ParseStreamID(t, "01HK154QBR6CK7JX737HQB4V58")
	evt1 := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", streamID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", streamID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, id.NewStreamRef(id.StreamType("User"), streamID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", events, 3)
}

func TestMemoryStore_AppendBatch_AppendsToExisting(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	streamID := idtest.ParseStreamID(t, "01HK154RB0WD5V767Z27XMXRX0")
	evt1 := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)
	_ = store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt1},
		0,
	)

	evt2 := eventtest.QuickEvent("UserUpdated", streamID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", streamID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, id.NewStreamRef(id.StreamType("User"), streamID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events total", events, 3)
}

func TestMemoryStore_AppendBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	streamID := idtest.ParseStreamID(t, "01HK154SA8Y7AMZCYV919GE46K")
	evt := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)

	err := store.AppendBatch(
		context.Background(),
		id.NewStreamRef("User", streamID),
		[]event.Event{evt},
	)
	if err == nil {
		t.Error("expected store closed error")
	}
}
