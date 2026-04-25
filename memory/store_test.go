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

	aggID := id.MustParseAggregateID("user-1")
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

	aggID := id.MustParseAggregateID("user-1")
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

	_, err := store.Load(ctx, "User", id.MustParseAggregateID("nonexistent"))
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("user-2")
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

	aggID := id.MustParseAggregateID("user-3")
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

	aggID := id.MustParseAggregateID("user-1")
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

	_, err := store.Load(ctx, "User", id.MustParseAggregateID("user-1"))
	if err == nil {
		t.Error("expected store closed error on Load")
	}
}

func TestMemoryStore_ClosedLoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	_, err := store.LoadFromVersion(ctx, "User", id.MustParseAggregateID("user-1"), 0)
	if err == nil {
		t.Error("expected store closed error on LoadFromVersion")
	}
}

func TestMemoryStore_ClosedDelete(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	err := store.Delete(ctx, "User", id.MustParseAggregateID("user-1"))
	if err == nil {
		t.Error("expected store closed error on Delete")
	}
}

func TestMemoryStore_LoadFromVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_, err := store.LoadFromVersion(ctx, "User", id.MustParseAggregateID("nonexistent"), 0)
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion_AtEnd(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("user-4")
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

	aggID := id.MustParseAggregateID("user-batch")
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

	aggID := id.MustParseAggregateID("user-batch2")
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

	aggID := id.MustParseAggregateID("user-batch3")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 0, nil)

	err := store.AppendBatch(context.Background(), "User", aggID, []event.Event{evt})
	if err == nil {
		t.Error("expected store closed error")
	}
}
