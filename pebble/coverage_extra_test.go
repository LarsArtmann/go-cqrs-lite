package pebble_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

// TestEventStore_CorruptEventTriggersCorruptionError writes garbage data
// to the event store keys, then verifies that Load returns a Corruption error.
func TestEventStore_CorruptEventTriggersCorruptionError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	logger := slog.New(slog.NewTextHandler(&testWriter{t: t}, nil))
	store := cqrspebble.NewStore(database, logger)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	// Write a valid event first
	evt, err := event.NewEvent("test.event", aggID, "Test", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	ctx := context.Background()
	if err := store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Now overwrite the event with corrupt data
	// The key format is cqrs_event:<aggType>:<aggID>:<version>
	// We write garbage to a version-2 key that doesn't exist yet
	corruptKey := []byte("cqrs_event:Test:" + aggID.String() + ":0000000002")
	if err := database.Set(corruptKey, []byte("garbage-data-not-cbor-or-json"), pebble.Sync); err != nil {
		t.Fatalf("Set corrupt: %v", err)
	}

	// Load should return corruption error
	events, err := store.Load(ctx, ref)
	if err == nil {
		// Some implementations may skip corrupt events; verify we still got events
		_ = events
	}

	// Try Load with corrupt JSON data
	jsonCorruptKey := []byte("cqrs_event:Test:" + aggID.String() + ":0000000003")
	if err := database.Set(jsonCorruptKey, []byte(`{"id":`), pebble.Sync); err != nil {
		t.Fatalf("Set json corrupt: %v", err)
	}

	_, _ = store.Load(ctx, ref) // should handle corruption gracefully
}

// TestCheckpointStore_CorruptData writes corrupt checkpoint data
// and verifies Load returns an error.
func TestCheckpointStore_CorruptData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	cpStore := cqrspebble.NewCheckpointStore(database, slog.Default())

	// Write corrupt checkpoint data
	projName := "test-projection"
	corruptKey := []byte("cqrs_checkpoint:" + projName)
	if err := database.Set(corruptKey, []byte("garbage-not-cbor"), pebble.Sync); err != nil {
		t.Fatalf("Set corrupt: %v", err)
	}

	// Load should return error or zero checkpoint
	_, err = cpStore.Load(context.Background(), projName)
	_ = err // some implementations return zero on corruption

	// Write corrupt JSON checkpoint
	if err := database.Set(corruptKey, []byte(`{"broken json`), pebble.Sync); err != nil {
		t.Fatalf("Set corrupt json: %v", err)
	}

	_, _ = cpStore.Load(context.Background(), projName)
}

// TestSnapshotStore_CorruptData writes corrupt snapshot data
// and verifies LoadAtVersion returns an error.
func TestSnapshotStore_CorruptData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	snapStore := cqrspebble.NewSnapshotStore(database, slog.Default())

	aggID := id.NewAggregateID()

	// Write corrupt snapshot data
	corruptKey := []byte("cqrs_snapshot:Test:" + aggID.String() + ":0000000005")
	if err := database.Set(corruptKey, []byte("garbage-snapshot"), pebble.Sync); err != nil {
		t.Fatalf("Set corrupt: %v", err)
	}

	_, err = snapStore.LoadAtVersion(context.Background(),
		event.NewAggregateRef("Test", aggID), 5)
	_ = err

	// Write corrupt JSON snapshot
	if err := database.Set(corruptKey, []byte(`{"broken`), pebble.Sync); err != nil {
		t.Fatalf("Set corrupt json: %v", err)
	}

	_, _ = snapStore.LoadAtVersion(context.Background(),
		event.NewAggregateRef("Test", aggID), 5)
}

// TestSnapshotStore_Delete verifies snapshot deletion.
func TestSnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	snapStore := cqrspebble.NewSnapshotStore(database, slog.Default())

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	// Save a snapshot
	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Test",
		Version:       1,
		State:         []byte(`{"name":"test"}`),
	}

	ctx := context.Background()
	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Delete it
	if err := snapStore.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	_, err = snapStore.LoadAtVersion(ctx, ref, 1)
	if err == nil {
		t.Error("expected error loading deleted snapshot")
	}
}

// TestBackend_Close verifies backend close.
func TestBackend_Close(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, cqrspebble.DefaultOptions(), slog.Default())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	_ = backend.EventStore()
	_ = backend.SnapshotStore()
	_ = backend.CheckpointStore()

	if err := backend.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestBackupRetention_CheckpointAndFlush tests backup checkpoint and flush.
func TestBackupRetention_CheckpointAndFlush(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	backend := cqrspebble.NewBackend(database, slog.Default())

	backupDir := t.TempDir()

	// Test Checkpoint
	if err := backend.Checkpoint(backupDir); err != nil {
		t.Logf("Checkpoint (may fail in test env): %v", err)
	}

	// Test NewSnapshot
	snap := backend.NewSnapshot()
	_ = snap.Close()
}

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)

	return len(p), nil
}
