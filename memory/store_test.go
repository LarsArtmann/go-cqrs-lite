package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt1 := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	evt2 := testhelpers.QuickEvent("UserUpdated", aggID, "User", 1, nil)

	err := store.Save(ctx, "User", aggID, []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Save(ctx, "User", aggID, []event.Event{evt2}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "events", events, 2)
}


func TestMemoryStore_VersionConflict(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)

	err := store.Save(ctx, "User", aggID, []event.Event{evt}, 5)
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestMemoryStore_AggregateNotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Load(ctx, "User", id.MustParseAggregateID("01HK154KER4E8AJ20Q4JD5TJ1E"))
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154ME034FVHK95R554AKSE")
	evt1 := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	evt2 := testhelpers.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := testhelpers.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	_ = store.Save(ctx, "User", aggID, []event.Event{evt1, evt2, evt3}, 0)

	events, err := store.LoadFromVersion(ctx, "User", aggID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "events from version", events, 2)
}


func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154ND8R5WR0KARTF6H4S1B")
	evt := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	_ = store.Save(ctx, "User", aggID, []event.Event{evt}, 0)

	err := store.Delete(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.Load(ctx, "User", aggID)
	if err == nil {
		t.Error("expected aggregate not found after delete")
	}
}

func TestMemoryStore_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)

	err := store.Save(ctx, "User", aggID, []event.Event{evt}, 0)
	if err == nil {
		t.Error("expected store closed error")
	}
}

func TestMemoryStore_ClosedLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	_, err := store.Load(ctx, "User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))
	if err == nil {
		t.Error("expected store closed error on Load")
	}
}

func TestMemoryStore_ClosedLoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	_, err := store.LoadFromVersion(ctx, "User", aggID, 0)
	if err == nil {
		t.Error("expected store closed error on LoadFromVersion")
	}
}

func TestMemoryStore_ClosedDelete(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	err := store.Delete(ctx, "User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))
	if err == nil {
		t.Error("expected store closed error on Delete")
	}
}

func TestMemoryStore_LoadFromVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154KER4E8AJ20Q4JD5TJ1E")

	_, err := store.LoadFromVersion(ctx, "User", aggID, 0)
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion_AtEnd(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154PCGXJ80RFXRASTMSSK0")
	evt := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	_ = store.Save(ctx, "User", aggID, []event.Event{evt}, 0)

	events, err := store.LoadFromVersion(ctx, "User", aggID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "events past end", events, 0)
}

func TestMemoryStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154QBR6CK7JX737HQB4V58")
	evt1 := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	evt2 := testhelpers.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := testhelpers.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "events", events, 3)
}

func TestMemoryStore_AppendBatch_AppendsToExisting(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154RB0WD5V767Z27XMXRX0")
	evt1 := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)
	_ = store.Save(ctx, "User", aggID, []event.Event{evt1}, 0)

	evt2 := testhelpers.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := testhelpers.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(ctx, "User", aggID, []event.Event{evt2, evt3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "events total", events, 3)
}

func TestMemoryStore_AppendBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK154SA8Y7AMZCYV919GE46K")
	evt := testhelpers.QuickEvent("UserCreated", aggID, "User", 0, nil)

	err := store.AppendBatch(context.Background(), "User", aggID, []event.Event{evt})
	if err == nil {
		t.Error("expected store closed error")
	}
}
