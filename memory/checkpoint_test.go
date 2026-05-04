package memory

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestMemoryCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := NewCheckpointStore()
	ctx := context.Background()

	eventID := id.NewEventID()

	err := store.Save(ctx, "my-projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "my-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != eventID {
		t.Errorf("Load = %v, want %v", loaded, eventID)
	}
}

func TestMemoryCheckpointStore_Load_Empty(t *testing.T) {
	t.Parallel()

	store := NewCheckpointStore()
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

	store := NewCheckpointStore()
	ctx := context.Background()

	first := id.NewEventID()
	second := id.NewEventID()

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

	if loaded != second {
		t.Errorf("Load = %v, want %v (second save)", loaded, second)
	}
}

func TestMemoryCheckpointStore_Close(t *testing.T) {
	t.Parallel()

	store := NewCheckpointStore()

	err := store.Close()
	if err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}
