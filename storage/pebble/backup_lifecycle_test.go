package pebble_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

// TestBackupRestore_FullLifecycle verifies that a Pebble checkpoint (backup)
// preserves data across ALL store types (events, snapshots, checkpoints) and
// that a restored backend is fully functional for continued operations.
func TestBackupRestore_FullLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// ── Phase 1: Set up source backend with data across all stores ──

	source, err := pebble.Open(t.TempDir(), pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open source: %v", err)
	}

	defer func() { _ = source.Close() }()

	eventStore := source.EventStore()
	snapStore := source.SnapshotStore()
	cpStore := source.CheckpointStore()

	streamA := id.NewStreamID()
	streamB := id.NewStreamID()
	refA := id.NewStreamRef("User", streamA)
	refB := id.NewStreamRef("Order", streamB)

	// Write events to two streams.
	evtA1, _ := event.NewEvent("UserCreated", streamA, "User", 1, []byte(`{"name":"alice"}`))
	if err := eventStore.Save(ctx, refA, []event.Event{evtA1}, 0); err != nil {
		t.Fatalf("Save A1: %v", err)
	}

	evtB1, _ := event.NewEvent("OrderPlaced", streamB, "Order", 1, []byte(`{"total":42}`))
	evtB2, _ := event.NewEvent("OrderShipped", streamB, "Order", 2, []byte(`{"tracking":"XYZ"}`))
	if err := eventStore.Save(ctx, refB, []event.Event{evtB1, evtB2}, 0); err != nil {
		t.Fatalf("Save B1,B2: %v", err)
	}

	// Write a snapshot.
	snap := snapshot.Snapshot{
		StreamID:   streamA,
		StreamType: "User",
		Version:    1,
		State:      []byte(`{"name":"alice","version":1}`),
	}
	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	// Write a checkpoint.
	cpEventID := id.NewEventID()
	if err := cpStore.Save(ctx, "users-projection", event.Checkpoint{
		EventID:     cpEventID,
		ProcessedAt: evtA1.OccurredAt(),
	}); err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	// ── Phase 2: Create backup ──

	backupDir := t.TempDir() + "/backup"

	if err := source.Flush(); err != nil {
		t.Fatalf("Flush before checkpoint: %v", err)
	}

	if err := source.Checkpoint(backupDir); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}

	// ── Phase 3: Write MORE data after backup (must NOT appear in restore) ──

	evtA2, _ := event.NewEvent("UserUpdated", streamA, "User", 2, []byte(`{"name":"alice2"}`))
	if err := eventStore.Save(ctx, refA, []event.Event{evtA2}, 1); err != nil {
		t.Fatalf("Save A2 post-backup: %v", err)
	}

	if err := cpStore.Save(ctx, "users-projection", event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: evtA2.OccurredAt(),
	}); err != nil {
		t.Fatalf("Save checkpoint post-backup: %v", err)
	}

	// ── Phase 4: Restore from backup and verify ──

	restored, err := pebble.Open(backupDir, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open restored: %v", err)
	}

	defer func() { _ = restored.Close() }()

	// Events: stream A should have only evtA1 (pre-backup), not evtA2.
	loadedA, err := restored.EventStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("Load restored A: %v", err)
	}

	if len(loadedA) != 1 {
		t.Errorf("restored stream A: %d events, want 1 (pre-backup only)", len(loadedA))
	}

	// Events: stream B should have both events (pre-backup).
	loadedB, err := restored.EventStore().Load(ctx, refB)
	if err != nil {
		t.Fatalf("Load restored B: %v", err)
	}

	if len(loadedB) != 2 {
		t.Errorf("restored stream B: %d events, want 2", len(loadedB))
	}

	// Snapshot: should be present from pre-backup.
	loadedSnap, err := restored.SnapshotStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("Load restored snapshot: %v", err)
	}

	if loadedSnap.Version.Int() != 1 {
		t.Errorf("restored snapshot version = %d, want 1", loadedSnap.Version.Int())
	}

	// Checkpoint: should have cpEventID (pre-backup), NOT the post-backup value.
	loadedCP, err := restored.CheckpointStore().Load(ctx, "users-projection")
	if err != nil {
		t.Fatalf("Load restored checkpoint: %v", err)
	}

	if loadedCP.EventID != cpEventID {
		t.Errorf("restored checkpoint EventID = %s, want %s (pre-backup)", loadedCP.EventID, cpEventID)
	}

	// ── Phase 5: Restored backend is functional for new writes ──

	evtA3, _ := event.NewEvent("UserDeleted", streamA, "User", 2, []byte(`{}`))
	if err := restored.EventStore().Save(ctx, refA, []event.Event{evtA3}, 1); err != nil {
		t.Fatalf("Save to restored backend: %v", err)
	}

	loadedA2, err := restored.EventStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("Load after new write: %v", err)
	}

	if len(loadedA2) != 2 {
		t.Errorf("after new write: %d events, want 2", len(loadedA2))
	}
}

// TestBackupRestore_IncrementalCheckpoints verifies that multiple sequential
// checkpoints capture the state at each point in time.
func TestBackupRestore_IncrementalCheckpoints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	source, err := pebble.Open(t.TempDir(), pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	defer func() { _ = source.Close() }()

	store := source.EventStore()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	// Write event 1, checkpoint.
	evt1, _ := event.NewEvent("Incremented", streamID, "Counter", 1, []byte(`{}`))
	_ = store.Save(ctx, ref, []event.Event{evt1}, 0)
	_ = source.Flush()

	backup1 := t.TempDir() + "/b1"
	if err := source.Checkpoint(backup1); err != nil {
		t.Fatalf("Checkpoint 1: %v", err)
	}

	// Write event 2, checkpoint again.
	evt2, _ := event.NewEvent("Incremented", streamID, "Counter", 2, []byte(`{}`))
	_ = store.Save(ctx, ref, []event.Event{evt2}, 1)
	_ = source.Flush()

	backup2 := t.TempDir() + "/b2"
	if err := source.Checkpoint(backup2); err != nil {
		t.Fatalf("Checkpoint 2: %v", err)
	}

	// Restore backup 1: should have 1 event.
	r1, err := pebble.Open(backup1, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open backup1: %v", err)
	}

	defer func() { _ = r1.Close() }()

	loaded1, _ := r1.EventStore().Load(ctx, ref)
	if len(loaded1) != 1 {
		t.Errorf("backup1: %d events, want 1", len(loaded1))
	}

	// Restore backup 2: should have 2 events.
	r2, err := pebble.Open(backup2, pebble.DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("Open backup2: %v", err)
	}

	defer func() { _ = r2.Close() }()

	loaded2, _ := r2.EventStore().Load(ctx, ref)
	if len(loaded2) != 2 {
		t.Errorf("backup2: %d events, want 2", len(loaded2))
	}
}
