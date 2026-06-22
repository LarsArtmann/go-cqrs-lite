package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3/idtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

func TestMemoryStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := idtest.ParseAggregateID(t, "01HK154QBR6CK7JX737HQB4V58")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", events, 3)
}

func TestMemoryStore_AppendBatch_AppendsToExisting(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := idtest.ParseAggregateID(t, "01HK154RB0WD5V767Z27XMXRX0")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1},
		0,
	)

	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events total", events, 3)
}

func TestMemoryStore_AppendBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	aggID := idtest.ParseAggregateID(t, "01HK154SA8Y7AMZCYV919GE46K")
	evt := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		[]event.Event{evt},
	)
	if err == nil {
		t.Error("expected store closed error")
	}
}
