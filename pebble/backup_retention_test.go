package pebble_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/pebble/v2"
)

func TestBackend_Checkpoint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	backend, err := pebble.Open(dir, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	defer func() { _ = backend.Close() }()

	store := backend.EventStore()

	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = store.Save(context.Background(), event.NewAggregateRef("User", aggID),
		[]event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	checkpointDir := filepath.Join(t.TempDir(), "checkpoint")
	if err := backend.Checkpoint(checkpointDir); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}

	entries, err := os.ReadDir(checkpointDir)
	if err != nil {
		t.Fatalf("ReadDir checkpoint: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("checkpoint directory is empty")
	}

	restored, err := pebble.Open(checkpointDir, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open checkpoint: %v", err)
	}

	defer func() { _ = restored.Close() }()

	loaded, err := restored.EventStore().Load(context.Background(),
		event.NewAggregateRef("User", aggID))
	if err != nil {
		t.Fatalf("Load from checkpoint: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event in checkpoint, got %d", len(loaded))
	}
}

func TestBackend_NewSnapshot_ConsistentReads(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	backend, err := pebble.Open(dir, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	defer func() { _ = backend.Close() }()

	store := backend.EventStore()

	aggID := id.NewAggregateID()

	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	if err := store.Save(context.Background(), event.NewAggregateRef("User", aggID),
		[]event.Event{evt1}, 0); err != nil {
		t.Fatalf("Save evt1: %v", err)
	}

	if err := backend.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	snap := backend.NewSnapshot()

	defer func() { _ = snap.Close() }()

	evt2, _ := event.NewEvent("UserUpdated", aggID, "User", 2, []byte(`{}`))
	if err := store.Save(context.Background(), event.NewAggregateRef("User", aggID),
		[]event.Event{evt2}, 1); err != nil {
		t.Fatalf("Save evt2: %v", err)
	}

	liveEvents, err := store.Load(context.Background(),
		event.NewAggregateRef("User", aggID))
	if err != nil {
		t.Fatalf("Live Load: %v", err)
	}

	if len(liveEvents) != 2 {
		t.Fatalf("live read: expected 2 events, got %d", len(liveEvents))
	}

	iter, err := snap.NewIter(nil)
	if err != nil {
		t.Fatalf("snap.NewIter: %v", err)
	}

	defer func() { _ = iter.Close() }()

	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	// 1 event = 2 keys (cqrs_event + cqrs_journal).
	// Second event was written AFTER snapshot — its keys must be invisible.
	if count > 2 {
		t.Errorf("snapshot saw %d keys, expected at most 2 (one event before snapshot)", count)
	}
}
