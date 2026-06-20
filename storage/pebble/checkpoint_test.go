package pebble

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func newCheckpointStore(t *testing.T) *CheckpointStore {
	t.Helper()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return NewCheckpointStore(db, slog.Default())
}

func TestCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)
	ctx := context.Background()
	processedAt := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	checkpoint := event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: processedAt,
	}

	if err := store.Save(ctx, "user-projection", checkpoint); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "user-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.EventID != checkpoint.EventID {
		t.Errorf("EventID = %s, want %s", loaded.EventID, checkpoint.EventID)
	}

	if !loaded.ProcessedAt.Equal(processedAt) {
		t.Errorf("ProcessedAt = %v, want %v", loaded.ProcessedAt, processedAt)
	}
}

func TestCheckpointStore_Load_NotFound_ReturnsZero(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)

	loaded, err := store.Load(context.Background(), "never-saved")
	if err != nil {
		t.Fatalf("Load on missing projection should not error, got: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("expected zero checkpoint for missing projection, got %+v", loaded)
	}
}

func TestCheckpointStore_Save_Overwrites(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)
	ctx := context.Background()
	projection := "user-projection"

	first := event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: time.Now(),
	}
	if err := store.Save(ctx, projection, first); err != nil {
		t.Fatalf("Save first: %v", err)
	}

	second := event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: time.Now().Add(time.Second),
	}
	if err := store.Save(ctx, projection, second); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	loaded, err := store.Load(ctx, projection)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.EventID != second.EventID {
		t.Errorf("EventID = %s, want %s (latest)", loaded.EventID, second.EventID)
	}
}

func TestCheckpointStore_DistinctProjections(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)
	ctx := context.Background()

	cpA := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
	cpB := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}

	if err := store.Save(ctx, "projection-a", cpA); err != nil {
		t.Fatalf("Save A: %v", err)
	}

	if err := store.Save(ctx, "projection-b", cpB); err != nil {
		t.Fatalf("Save B: %v", err)
	}

	loadedA, err := store.Load(ctx, "projection-a")
	if err != nil {
		t.Fatalf("Load A: %v", err)
	}

	loadedB, err := store.Load(ctx, "projection-b")
	if err != nil {
		t.Fatalf("Load B: %v", err)
	}

	if loadedA.EventID != cpA.EventID {
		t.Errorf("projection A EventID = %s, want %s", loadedA.EventID, cpA.EventID)
	}

	if loadedB.EventID != cpB.EventID {
		t.Errorf("projection B EventID = %s, want %s", loadedB.EventID, cpB.EventID)
	}
}

func TestCheckpointStore_Save_EmptyName(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)

	err := store.Save(context.Background(), "", event.Checkpoint{})
	if err == nil {
		t.Fatal("expected error for empty projection name")
	}
}

func TestCheckpointStore_Load_EmptyName(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)

	_, err := store.Load(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty projection name")
	}
}

func TestCheckpointStore_NewStore_NilDB(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when calling NewCheckpointStore with nil db")
		}

		msg, ok := r.(string)
		if !ok || msg != "pebble: NewCheckpointStore called with nil db" {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	NewCheckpointStore(nil, slog.Default())
}

func TestCheckpointStore_Close_NoOp(t *testing.T) {
	t.Parallel()

	store := newCheckpointStore(t)

	if err := store.Close(); err != nil {
		t.Fatalf("Close should be no-op: %v", err)
	}

	// Verify still usable after Close (DB lifetime is caller-owned).
	err := store.Save(context.Background(), "post-close", event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Save after Close: %v", err)
	}
}

func TestCheckpointStore_SharedDB_WithEventStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eventStore := NewStore(db, slog.Default())
	cpStore := NewCheckpointStore(db, slog.Default())

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Issue", aggID)
	cfg := issueStoreConfig()

	evt := cfg.NewTestEvent(t, aggID, 1)
	if err := eventStore.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("event Save: %v", err)
	}

	checkpoint := event.Checkpoint{EventID: evt.ID(), ProcessedAt: time.Now()}
	if err := cpStore.Save(ctx, "issue-projection", checkpoint); err != nil {
		t.Fatalf("checkpoint Save: %v", err)
	}

	loaded, err := cpStore.Load(ctx, "issue-projection")
	if err != nil {
		t.Fatalf("checkpoint Load: %v", err)
	}

	if loaded.EventID != evt.ID() {
		t.Errorf("checkpoint EventID = %s, want %s", loaded.EventID, evt.ID())
	}

	events, err := eventStore.ReadAll(ctx)
	if err != nil {
		t.Fatalf("event ReadAll: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("events = %d, want 1 (checkpoint should not pollute journal)", len(events))
	}
}
