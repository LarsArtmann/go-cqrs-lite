package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestMemoryStore_LoadAll(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	evt1 := testhelpers.QuickEvent("UserCreated", agg1, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("OrderPlaced", agg2, "Order", 1, nil)

	err := store.Save(ctx, event.AggregateType("User"), agg1, []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("save user event: %v", err)
	}

	err = store.Save(ctx, event.AggregateType("Order"), agg2, []event.Event{evt2}, 0)
	if err != nil {
		t.Fatalf("save order event: %v", err)
	}

	all, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("load all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestMemoryStore_LoadAll_Empty(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	all, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("load all empty: %v", err)
	}

	if len(all) != 0 {
		t.Fatalf("expected 0 events, got %d", len(all))
	}
}

func TestMemoryStore_LoadAll_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}

	_, err = store.LoadAll(ctx)
	if err == nil {
		t.Fatal("expected error on closed store")
	}
}

func TestMemoryCheckpointStore_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewCheckpointStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestMemoryOutboxStore_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryOutboxStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
}
