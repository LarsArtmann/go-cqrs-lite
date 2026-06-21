package pebble_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v2"
)

func TestEventStore_OptionAsyncWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default(), cqrspebble.WithAsyncWrites())
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("TestOpts", aggID)
	evt, err := event.NewEvent("test.event", aggID, "TestOpts", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.Save(
		context.Background(),
		ref,
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestCheckpointStore_Options(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	cpStore, err := cqrspebble.NewCheckpointStore(database, slog.Default(),
		cqrspebble.WithCheckpointAsyncWrites(),
		cqrspebble.WithCheckpointPrefix("custom_cp:"))

	ctx := context.Background()

	err = cpStore.Save(ctx, "test-proj", event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = cpStore.Load(ctx, "test-proj")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestSnapshotStore_Options(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	snapStore, err := cqrspebble.NewSnapshotStore(database, slog.Default(),
		cqrspebble.WithSnapshotAsyncWrites(),
		cqrspebble.WithSnapshotPrefix("custom_snap:"))

	ctx := context.Background()

	aggID := id.NewAggregateID()
	aggType := event.AggregateType("TestOpts")
	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       event.Version(1),
		State:         []byte(`{"state":"ok"}`),
		CreatedAt:     time.Now(),
	}

	err = snapStore.Save(ctx, snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	ref := event.NewAggregateRef(aggType, aggID)
	loaded, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != event.Version(1) {
		t.Fatalf("expected version 1, got %d", loaded.Version)
	}
}

func TestEventStore_SaveEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ref := event.NewAggregateRef("Empty", id.NewAggregateID())

	err = store.Save(context.Background(), ref, nil, event.Version(0))
	if err != nil {
		t.Fatalf("Save with zero events should return nil, got: %v", err)
	}
}

func TestEventStore_LoadNonExistent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ref := event.NewAggregateRef("NoExist", id.NewAggregateID())

	events, err := store.Load(context.Background(), ref)
	if err == nil && len(events) > 0 {
		t.Fatal("expected empty result or error for non-existent aggregate")
	}
}

func TestBackend_OpenError(t *testing.T) {
	t.Parallel()

	// Opening a pebble DB in /dev/null/x should fail
	_, err := cqrspebble.Open("/dev/null/impossible", &pebble.Options{}, slog.Default())
	if err == nil {
		t.Fatal("expected error opening pebble in impossible path")
	}
}

func TestEventStore_AppendBatchMultiple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("TestAppend", aggID)
	evt, _ := event.NewEvent("test.created", aggID, "TestAppend", event.Version(1), []byte(`{}`))
	_ = store.Save(ctx, ref, []event.Event{evt}, event.Version(0))

	evt2, _ := event.NewEvent("test.updated", aggID, "TestAppend", event.Version(2), []byte(`{}`))
	evt3, _ := event.NewEvent("test.deleted", aggID, "TestAppend", event.Version(3), []byte(`{}`))
	err = store.AppendBatch(ctx, ref, []event.Event{evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestCheckpointStore_EmptyName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	cpStore, err := cqrspebble.NewCheckpointStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	err = cpStore.Save(context.Background(), "", event.Checkpoint{})
	if err == nil {
		t.Fatal("expected error saving checkpoint with empty projection name")
	}
}

func TestEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	baseTime := time.Now()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("TestLFV", aggID)

	for i := range 3 {
		evt, _ := event.NewEvent("test.event", aggID, "TestLFV", event.Version(i+1),
			[]byte(`{}`), event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Second)))
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	// Load from version 1 → should get events 2 and 3
	events, err := store.LoadFromVersion(ctx, ref, event.Version(1))
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events from version 1, got %d", len(events))
	}
}

func TestEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	baseTime := time.Now()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("TestLTT", aggID)

	for i := range 3 {
		evt, _ := event.NewEvent("test.event", aggID, "TestLTT", event.Version(i+1),
			[]byte(`{}`), event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Second)))
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	// Load up to the 2nd event's timestamp
	events, err := store.LoadToTimestamp(ctx, ref, baseTime.Add(1500*time.Millisecond))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events up to timestamp, got %d", len(events))
	}
}

func TestSnapshotStore_LoadNonExistent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	snapStore, err := cqrspebble.NewSnapshotStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ref := event.NewAggregateRef("NoSnap", id.NewAggregateID())

	_, err = snapStore.Load(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error loading non-existent snapshot")
	}
}
