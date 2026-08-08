package bbolt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// TestBackupRestore_FullLifecycle verifies that a bbolt backup (via tx.WriteTo)
// preserves data across ALL store types (events, snapshots, checkpoints) and
// that a restored backend is fully functional for continued operations.
func TestBackupRestore_FullLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// ── Phase 1: Set up source backend with data across all stores ──

	source, err := Open(filepath.Join(t.TempDir(), "source.db"), nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}

	defer deferClose(source)

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
		t.Fatalf("save A1: %v", err)
	}

	evtB1, _ := event.NewEvent("OrderPlaced", streamB, "Order", 1, []byte(`{"total":42}`))
	evtB2, _ := event.NewEvent("OrderShipped", streamB, "Order", 2, []byte(`{"tracking":"XYZ"}`))
	if err := eventStore.Save(ctx, refB, []event.Event{evtB1, evtB2}, 0); err != nil {
		t.Fatalf("save B1,B2: %v", err)
	}

	// Write a snapshot.
	snap := snapshot.Snapshot{
		StreamID:   streamA,
		StreamType: "User",
		Version:    1,
		State:      []byte(`{"name":"alice","version":1}`),
	}
	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Write a checkpoint.
	cpEventID := id.NewEventID()
	if err := cpStore.Save(ctx, "users-projection", event.Checkpoint{
		EventID:     cpEventID,
		ProcessedAt: evtA1.OccurredAt(),
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	// ── Phase 2: Create backup via bbolt tx.WriteTo ──

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	backupFile(t, source.DB(), backupPath)

	// ── Phase 3: Write MORE data after backup (must NOT appear in restore) ──

	evtA2, _ := event.NewEvent("UserUpdated", streamA, "User", 2, []byte(`{"name":"alice2"}`))
	if err := eventStore.Save(ctx, refA, []event.Event{evtA2}, 1); err != nil {
		t.Fatalf("save A2 post-backup: %v", err)
	}

	if err := cpStore.Save(ctx, "users-projection", event.Checkpoint{
		EventID:     id.NewEventID(),
		ProcessedAt: evtA2.OccurredAt(),
	}); err != nil {
		t.Fatalf("save checkpoint post-backup: %v", err)
	}

	// ── Phase 4: Restore from backup and verify ──

	restored, err := Open(backupPath, nil)
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}

	defer deferClose(restored)

	// Events: stream A should have only evtA1 (pre-backup), not evtA2.
	loadedA, err := restored.EventStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("load restored A: %v", err)
	}

	if len(loadedA) != 1 {
		t.Errorf("restored stream A: %d events, want 1 (pre-backup only)", len(loadedA))
	}

	// Events: stream B should have both events (pre-backup).
	loadedB, err := restored.EventStore().Load(ctx, refB)
	if err != nil {
		t.Fatalf("load restored B: %v", err)
	}

	if len(loadedB) != 2 {
		t.Errorf("restored stream B: %d events, want 2", len(loadedB))
	}

	// Snapshot: should be present from pre-backup.
	loadedSnap, err := restored.SnapshotStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("load restored snapshot: %v", err)
	}

	if loadedSnap.Version.Int() != 1 {
		t.Errorf("restored snapshot version = %d, want 1", loadedSnap.Version.Int())
	}

	// Checkpoint: should have cpEventID (pre-backup), NOT the post-backup value.
	loadedCP, err := restored.CheckpointStore().Load(ctx, "users-projection")
	if err != nil {
		t.Fatalf("load restored checkpoint: %v", err)
	}

	if loadedCP.EventID != cpEventID {
		t.Errorf(
			"restored checkpoint EventID = %s, want %s (pre-backup)",
			loadedCP.EventID,
			cpEventID,
		)
	}

	// ── Phase 5: Restored backend is functional for new writes ──

	evtA3, _ := event.NewEvent("UserDeleted", streamA, "User", 2, []byte(`{}`))
	if err := restored.EventStore().Save(ctx, refA, []event.Event{evtA3}, 1); err != nil {
		t.Fatalf("save to restored backend: %v", err)
	}

	loadedA2, err := restored.EventStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("load after new write: %v", err)
	}

	if len(loadedA2) != 2 {
		t.Errorf("after new write: %d events, want 2", len(loadedA2))
	}
}

// TestBackupRestore_IncrementalCheckpoints verifies that multiple sequential
// backups capture the state at each point in time.
func TestBackupRestore_IncrementalCheckpoints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	source, err := Open(filepath.Join(t.TempDir(), "source.db"), nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer deferClose(source)

	store := source.EventStore()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	// Write event 1, backup.
	evt1, _ := event.NewEvent("Incremented", streamID, "Counter", 1, []byte(`{}`))
	if err := store.Save(ctx, ref, []event.Event{evt1}, 0); err != nil {
		t.Fatalf("save evt1: %v", err)
	}

	backup1 := filepath.Join(t.TempDir(), "b1.db")
	backupFile(t, source.DB(), backup1)

	// Write event 2, backup again.
	evt2, _ := event.NewEvent("Incremented", streamID, "Counter", 2, []byte(`{}`))
	if err := store.Save(ctx, ref, []event.Event{evt2}, 1); err != nil {
		t.Fatalf("save evt2: %v", err)
	}

	backup2 := filepath.Join(t.TempDir(), "b2.db")
	backupFile(t, source.DB(), backup2)

	// Restore backup 1: should have 1 event.
	r1, err := Open(backup1, nil)
	if err != nil {
		t.Fatalf("open backup1: %v", err)
	}

	defer deferClose(r1)

	loaded1, err := r1.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("load backup1: %v", err)
	}

	if len(loaded1) != 1 {
		t.Errorf("backup1: %d events, want 1", len(loaded1))
	}

	// Restore backup 2: should have 2 events.
	r2, err := Open(backup2, nil)
	if err != nil {
		t.Fatalf("open backup2: %v", err)
	}

	defer deferClose(r2)

	loaded2, err := r2.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("load backup2: %v", err)
	}

	if len(loaded2) != 2 {
		t.Errorf("backup2: %d events, want 2", len(loaded2))
	}
}

func backupFile(t *testing.T, db *bolt.DB, destPath string) {
	t.Helper()

	f, err := os.Create(destPath)
	if err != nil {
		t.Fatalf("create backup file: %v", err)
	}

	defer f.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(f)
		return err
	}); err != nil {
		t.Fatalf("backup: %v", err)
	}
}
