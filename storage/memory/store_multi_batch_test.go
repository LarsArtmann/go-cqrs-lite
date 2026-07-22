package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	idtest "github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_SaveMultiBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggA := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	aggB := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX2VLR96")
	aggC := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX3VLR97")

	entries := []event.MultiBatchEntry{
		{
			Ref: id.NewAggregateRef("User", aggA),
			Events: []event.Event{
				eventtest.QuickEvent("UserCreated", aggA, "User", 1, nil),
			},
		},
		{
			Ref: id.NewAggregateRef("User", aggB),
			Events: []event.Event{
				eventtest.QuickEvent("UserCreated", aggB, "User", 1, nil),
				eventtest.QuickEvent("UserUpdated", aggB, "User", 2, nil),
			},
		},
		{
			Ref: id.NewAggregateRef("Order", aggC),
			Events: []event.Event{
				eventtest.QuickEvent("OrderPlaced", aggC, "Order", 1, nil),
			},
		},
	}

	err := store.SaveMultiBatch(ctx, entries)
	if err != nil {
		t.Fatalf("SaveMultiBatch failed: %v", err)
	}

	eventsA, err := store.Load(ctx, id.NewAggregateRef("User", aggA))
	if err != nil {
		t.Fatalf("Load A failed: %v", err)
	}

	eventtest.AssertLen(t, "events A", eventsA, 1)

	eventsB, err := store.Load(ctx, id.NewAggregateRef("User", aggB))
	if err != nil {
		t.Fatalf("Load B failed: %v", err)
	}

	eventtest.AssertLen(t, "events B", eventsB, 2)

	eventsC, err := store.Load(ctx, id.NewAggregateRef("Order", aggC))
	if err != nil {
		t.Fatalf("Load C failed: %v", err)
	}

	eventtest.AssertLen(t, "events C", eventsC, 1)

	journal := store // MemoryStore implements event.Journal

	all, err := journal.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	eventtest.AssertLen(t, "global log", all, 4)
}

func TestMemoryStore_SaveMultiBatch_Empty(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	err := store.SaveMultiBatch(ctx, nil)
	if err != nil {
		t.Fatalf("SaveMultiBatch with nil entries should succeed: %v", err)
	}

	err = store.SaveMultiBatch(ctx, []event.MultiBatchEntry{})
	if err != nil {
		t.Fatalf("SaveMultiBatch with empty entries should succeed: %v", err)
	}

	entries := []event.MultiBatchEntry{
		{Ref: id.NewAggregateRef("User", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")), Events: nil},
	}

	err = store.SaveMultiBatch(ctx, entries)
	if err != nil {
		t.Fatalf("SaveMultiBatch with nil events should succeed: %v", err)
	}
}

func TestMemoryStore_SaveMultiBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	aggA := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	err := store.SaveMultiBatch(context.Background(), []event.MultiBatchEntry{
		{
			Ref:    id.NewAggregateRef("User", aggA),
			Events: []event.Event{eventtest.QuickEvent("UserCreated", aggA, "User", 1, nil)},
		},
	})
	if err == nil {
		t.Fatal("expected error from SaveMultiBatch on closed store")
	}
}
