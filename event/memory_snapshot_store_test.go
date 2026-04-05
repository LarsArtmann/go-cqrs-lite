package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestMemorySnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := event.Snapshot{
		AggregateID:   "order-1",
		AggregateType: "Order",
		Version:       event.Version(5),
		State:         []byte(`{"status":"shipped"}`),
		CreatedAt:     time.Now(),
	}

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", "order-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.AggregateID != snapshot.AggregateID {
		t.Errorf("expected aggregate ID %s, got %s", snapshot.AggregateID, loaded.AggregateID)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("expected version 5, got %d", loaded.Version.Int())
	}

	if string(loaded.State) != `{"status":"shipped"}` {
		t.Errorf("unexpected state: %s", string(loaded.State))
	}
}

func TestMemorySnapshotStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()

	_, err := store.Load(context.Background(), "Order", "nonexistent")
	if err == nil {
		t.Error("expected snapshot not found error")
	}
}

func TestMemorySnapshotStore_Save_IgnoresOlderVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV5 := event.Snapshot{
		AggregateID:   "order-2",
		AggregateType: "Order",
		Version:       event.Version(5),
		State:         []byte(`{"status":"shipped"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshotV5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV3 := event.Snapshot{
		AggregateID:   "order-2",
		AggregateType: "Order",
		Version:       event.Version(3),
		State:         []byte(`{"status":"placed"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshotV3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", "order-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("expected version 5, got %d (should not downgrade)", loaded.Version.Int())
	}
}

func TestMemorySnapshotStore_Save_UpdatesNewerVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV3 := event.Snapshot{
		AggregateID:   "order-3",
		AggregateType: "Order",
		Version:       event.Version(3),
		State:         []byte(`{"status":"placed"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshotV3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV7 := event.Snapshot{
		AggregateID:   "order-3",
		AggregateType: "Order",
		Version:       event.Version(7),
		State:         []byte(`{"status":"delivered"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshotV7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", "order-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Version.Int() != 7 {
		t.Errorf("expected version 7, got %d", loaded.Version.Int())
	}
}

func TestMemorySnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := event.Snapshot{
		AggregateID:   "order-4",
		AggregateType: "Order",
		Version:       event.Version(5),
		State:         []byte(`{"status":"shipped"}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("at exact version", func(t *testing.T) {
		t.Parallel()

		loaded, err := store.LoadAtVersion(ctx, "Order", "order-4", event.Version(5))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if loaded.Version.Int() != 5 {
			t.Errorf("expected version 5, got %d", loaded.Version.Int())
		}
	})

	t.Run("after snapshot version", func(t *testing.T) {
		t.Parallel()

		loaded, err := store.LoadAtVersion(ctx, "Order", "order-4", event.Version(10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if loaded.Version.Int() != 5 {
			t.Errorf("expected version 5, got %d", loaded.Version.Int())
		}
	})

	t.Run("before snapshot version", func(t *testing.T) {
		t.Parallel()

		_, err := store.LoadAtVersion(ctx, "Order", "order-4", event.Version(3))
		if err == nil {
			t.Error("expected snapshot not found for version before snapshot")
		}
	})
}

func TestMemorySnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := event.Snapshot{
		AggregateID:   "order-5",
		AggregateType: "Order",
		Version:       event.Version(1),
		State:         []byte(`{}`),
		CreatedAt:     time.Now(),
	}

	if err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete(ctx, "Order", "order-5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := store.Load(ctx, "Order", "order-5")
	if err == nil {
		t.Error("expected snapshot not found after delete")
	}
}
