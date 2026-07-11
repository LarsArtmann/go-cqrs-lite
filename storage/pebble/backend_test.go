package pebble_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

func TestBackend_OpenAndClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := backend.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestBackend_GracefulClose(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := backend.GracefulClose(ctx); err != nil {
			t.Fatalf("GracefulClose failed: %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}

		// Immediately cancelled context — Pebble Close is fast, so this
		// tests the select path without guaranteeing a race. The key
		// assertion is that GracefulClose doesn't panic or deadlock.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = backend.GracefulClose(ctx) // may return nil or ctx.Err()
	})
}

func TestBackend_FullStack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	eventStore := backend.EventStore()
	snapStore := backend.SnapshotStore()
	cpStore := backend.CheckpointStore()

	if eventStore == nil || snapStore == nil || cpStore == nil {
		t.Fatal("one or more stores returned nil")
	}

	// Save an event
	aggID := id.NewAggregateID()
	aggType := id.AggregateType("User")
	ref := id.NewAggregateRef(aggType, aggID)
	evt, err := event.NewEvent("user.created", aggID, aggType, event.Version(1),
		[]byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if err := eventStore.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load it back
	loaded, err := eventStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	// Save a snapshot
	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       event.Version(1),
		State:         []byte(`{"name":"alice"}`),
		CreatedAt:     time.Now(),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Snapshot Save failed: %v", err)
	}

	loadedSnap, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Snapshot Load failed: %v", err)
	}

	if loadedSnap.Version != event.Version(1) {
		t.Fatalf("expected snapshot version 1, got %d", loadedSnap.Version)
	}

	// Save a checkpoint
	checkpoint := event.Checkpoint{
		EventID:     evt.ID(),
		ProcessedAt: time.Now(),
	}

	if err := cpStore.Save(ctx, "test-projection", checkpoint); err != nil {
		t.Fatalf("Checkpoint Save failed: %v", err)
	}

	loadedCP, err := cpStore.Load(ctx, "test-projection")
	if err != nil {
		t.Fatalf("Checkpoint Load failed: %v", err)
	}

	if loadedCP.EventID != evt.ID() {
		t.Fatalf("expected checkpoint event ID %s, got %s", evt.ID(), loadedCP.EventID)
	}

	// Verify ReadAll (journal)
	allEvents, err := eventStore.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(allEvents) != 1 {
		t.Fatalf("expected 1 event in journal, got %d", len(allEvents))
	}
}

func TestBackend_NewBackend_WrapsExistingDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	backend, err := cqrspebble.NewBackend(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	if backend.EventStore() == nil || backend.SnapshotStore() == nil ||
		backend.CheckpointStore() == nil {
		t.Fatal("one or more stores returned nil")
	}
}

func TestBackend_ReadFrom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	eventStore := backend.EventStore()
	aggID := id.NewAggregateID()
	aggType := id.AggregateType("Issue")
	ref := id.NewAggregateRef(aggType, aggID)
	baseTime := time.Now()

	// Save 5 events
	events := make([]event.Event, 0, 5)

	for i := range 5 {
		evt, err := event.NewEvent(
			"IssueCreated", aggID, aggType, event.Version(i+1),
			[]byte(fmt.Sprintf(`{"title":"test-%d"}`, i+1)),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Second)),
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := eventStore.AppendBatch(ctx, ref, events); err != nil {
		t.Fatalf("AppendBatch failed: %v", err)
	}

	// ReadFrom after event #2, limit 2
	fromMid, err := eventStore.ReadFrom(ctx, events[1].ID(), 2)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	if len(fromMid) != 2 {
		t.Fatalf("expected 2 events from ReadFrom, got %d", len(fromMid))
	}
}
