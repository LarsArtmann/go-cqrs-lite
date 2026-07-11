package memory

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestMemoryCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()
	ctx := context.Background()

	checkpoint := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}

	err := store.Save(ctx, "my-projection", checkpoint)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "my-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.EventID != checkpoint.EventID {
		t.Errorf("Load.EventID = %v, want %v", loaded.EventID, checkpoint.EventID)
	}
}

func TestMemoryCheckpointStore_Load_Empty(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()
	ctx := context.Background()

	loaded, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("Load nonexistent = %v, want zero value", loaded)
	}
}

func TestMemoryCheckpointStore_Save_Overwrite(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()
	ctx := context.Background()

	first := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
	second := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}

	err := store.Save(ctx, "proj", first)
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}

	err = store.Save(ctx, "proj", second)
	if err != nil {
		t.Fatalf("Save second: %v", err)
	}

	loaded, err := store.Load(ctx, "proj")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.EventID != second.EventID {
		t.Errorf("Load.EventID = %v, want %v (second save)", loaded.EventID, second.EventID)
	}
}

func TestMemoryCheckpointStore_Close(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()

	err := store.Close()
	if err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}
