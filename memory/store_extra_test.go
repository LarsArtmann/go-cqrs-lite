package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

//nolint:dupl // backward-compat test mirrors ReadAll
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

//nolint:dupl // backward-compat test mirrors ReadAll_Empty
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

//nolint:dupl // backward-compat test mirrors ReadAll_Closed
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

	store := memory.NewMemoryCheckpointStore()

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

//nolint:dupl // mirrors LoadAll backward-compat test
func TestMemoryStore_ReadAll(t *testing.T) {
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

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestMemoryStore_ReadAll_Empty(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read all empty: %v", err)
	}

	if len(all) != 0 {
		t.Fatalf("expected 0 events, got %d", len(all))
	}
}

func TestMemoryStore_ReadAll_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}

	_, err = store.ReadAll(ctx)
	if err == nil {
		t.Fatal("expected error on closed store")
	}
}

//nolint:dupl // mirrors LoadAllFromPosition backward-compat test
func TestMemoryStore_ReadFrom(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	now := time.Now()
	evt1, _ := event.NewEvent(
		"Created",
		aggID1,
		"User",
		1,
		nil,
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"Created",
		aggID2,
		"Order",
		1,
		nil,
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent("Updated", aggID1, "User", 1, nil, event.WithOccurredAt(now))

	_ = store.AppendBatch(ctx, "User", aggID1, []event.Event{evt1, evt3})
	_ = store.AppendBatch(ctx, "Order", aggID2, []event.Event{evt2})

	all, err := store.ReadAll(ctx)
	testhelpers.AssertNoError(t, err, "ReadAll")
	testhelpers.AssertLen(t, "all events", all, 3)

	fromPos, err := store.ReadFrom(ctx, evt1.ID(), 1)
	testhelpers.AssertNoError(t, err, "ReadFrom")
	testhelpers.AssertLen(t, "from position", fromPos, 1)
}

func TestMemoryStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt})

	events, err := store.ReadFrom(ctx, id.EventID{}, 10)
	testhelpers.AssertNoError(t, err, "ReadFrom with zero ID")
	testhelpers.AssertLen(t, "events", events, 1)
}

func TestMemoryStore_ReadFrom_WithLimit(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	for i := range 5 {
		evt := testhelpers.QuickEvent("Created", aggID, "User", event.Version(i+1), nil)
		_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt})
	}

	events, err := store.ReadFrom(ctx, id.EventID{}, 3)
	testhelpers.AssertNoError(t, err, "ReadFrom with limit")
	testhelpers.AssertLen(t, "events", events, 3)
}

func TestMemoryStore_ReadFrom_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.ReadFrom(context.Background(), id.EventID{}, 10)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}
