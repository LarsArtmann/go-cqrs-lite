package memory

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func TestCheckpointStore_SaveAfterClose(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()

	if err := store.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	err := store.Save(t.Context(), "test-projection", event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: time.Now(),
	})
	if err == nil {
		t.Error("Save() after Close() should return error")
	}
}

func TestCheckpointStore_LoadAfterClose(t *testing.T) {
	t.Parallel()

	store := NewMemoryCheckpointStore()

	if err := store.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	_, err := store.Load(t.Context(), "test-projection")
	if err == nil {
		t.Error("Load() after Close() should return error")
	}
}
