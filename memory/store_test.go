package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
	evt2, _ := event.NewEvent("UserUpdated", aggID, "User", 1, nil)

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

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestMemoryStore_VersionConflict(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)

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
	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
	evt2, _ := event.NewEvent("UserUpdated", aggID, "User", 1, nil)
	evt3, _ := event.NewEvent("UserDeleted", aggID, "User", 2, nil)

	_ = store.Save(ctx, "User", aggID, []event.Event{evt1, evt2, evt3}, 0)

	events, err := store.LoadFromVersion(ctx, "User", aggID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events from version 1, got %d", len(events))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154ND8R5WR0KARTF6H4S1B")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
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
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)

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
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
	_ = store.Save(ctx, "User", aggID, []event.Event{evt}, 0)

	events, err := store.LoadFromVersion(ctx, "User", aggID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events past end, got %d", len(events))
	}
}

func TestMemoryStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154QBR6CK7JX737HQB4V58")
	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
	evt2, _ := event.NewEvent("UserUpdated", aggID, "User", 1, nil)
	evt3, _ := event.NewEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestMemoryStore_AppendBatch_AppendsToExisting(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154RB0WD5V767Z27XMXRX0")
	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)
	_ = store.Save(ctx, "User", aggID, []event.Event{evt1}, 0)

	evt2, _ := event.NewEvent("UserUpdated", aggID, "User", 1, nil)
	evt3, _ := event.NewEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(ctx, "User", aggID, []event.Event{evt2, evt3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events total, got %d", len(events))
	}
}

func TestMemoryStore_AppendBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK154SA8Y7AMZCYV919GE46K")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)

	err := store.AppendBatch(context.Background(), "User", aggID, []event.Event{evt})
	if err == nil {
		t.Error("expected store closed error")
	}
}
