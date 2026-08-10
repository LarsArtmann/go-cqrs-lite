// Package backuptest provides a shared backup-lifecycle test suite that any
// storage backend can plug into via a thin Factory adapter. Eliminates the
// near-identical backup_lifecycle_test.go files that previously existed in
// storage/bbolt and storage/pebble.
package backuptest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// Backend is the contract a storage backend must satisfy to run the
// backup-lifecycle test suite. Each backend's concrete Backend type is wrapped
// in a thin adapter that returns interface-typed stores.
type Backend interface {
	EventStore() event.Store
	SnapshotStore() snapshot.SnapshotStore
	CheckpointStore() event.CheckpointStore
	Close() error
}

// Factory knows how to create, back up, and restore a specific backend.
// Each storage package constructs a Factory and passes it to the Run* functions.
type Factory struct {
	New     func(t *testing.T) Backend
	Backup  func(t *testing.T, src Backend, destPath string)
	Restore func(t *testing.T, backupPath string) Backend
}

// RunFullLifecycle verifies that a backup preserves data across ALL store types
// (events, snapshots, checkpoints) and that a restored backend is fully
// functional for continued operations.
func RunFullLifecycle(t *testing.T, f Factory) {
	t.Parallel()

	ctx := context.Background()

	source := f.New(t)
	t.Cleanup(func() { _ = source.Close() })

	streamA := id.NewStreamID()
	streamB := id.NewStreamID()
	refA := id.NewStreamRef("User", streamA)
	refB := id.NewStreamRef("Order", streamB)

	evtA1, _ := event.NewEvent("UserCreated", streamA, "User", 1, []byte(`{"name":"alice"}`))
	must(t, "save A1", source.EventStore().Save(ctx, refA, []event.Event{evtA1}, 0))

	evtB1, _ := event.NewEvent("OrderPlaced", streamB, "Order", 1, []byte(`{"total":42}`))
	evtB2, _ := event.NewEvent("OrderShipped", streamB, "Order", 2, []byte(`{"tracking":"XYZ"}`))
	must(t, "save B1,B2", source.EventStore().Save(ctx, refB, []event.Event{evtB1, evtB2}, 0))

	snap := snapshot.Snapshot{
		StreamID:   streamA,
		StreamType: "User",
		Version:    1,
		State:      []byte(`{"name":"alice","version":1}`),
	}
	must(t, "save snapshot", source.SnapshotStore().Save(ctx, snap))

	cpEventID := id.NewEventID()
	must(
		t,
		"save checkpoint",
		source.CheckpointStore().Save(ctx, "users-projection", event.Checkpoint{
			EventID:     cpEventID,
			ProcessedAt: evtA1.OccurredAt(),
		}),
	)

	backupPath := filepath.Join(t.TempDir(), "backup")
	f.Backup(t, source, backupPath)

	evtA2, _ := event.NewEvent("UserUpdated", streamA, "User", 2, []byte(`{"name":"alice2"}`))
	must(t, "save A2 post-backup", source.EventStore().Save(ctx, refA, []event.Event{evtA2}, 1))

	must(
		t,
		"save checkpoint post-backup",
		source.CheckpointStore().Save(ctx, "users-projection", event.Checkpoint{
			EventID:     id.NewEventID(),
			ProcessedAt: evtA2.OccurredAt(),
		}),
	)

	restored := f.Restore(t, backupPath)
	t.Cleanup(func() { _ = restored.Close() })

	loadedA, err := restored.EventStore().Load(ctx, refA)
	must(t, "load restored A", err)
	if len(loadedA) != 1 {
		t.Errorf("restored stream A: %d events, want 1 (pre-backup only)", len(loadedA))
	}

	loadedB, err := restored.EventStore().Load(ctx, refB)
	must(t, "load restored B", err)
	if len(loadedB) != 2 {
		t.Errorf("restored stream B: %d events, want 2", len(loadedB))
	}

	loadedSnap, err := restored.SnapshotStore().Load(ctx, refA)
	must(t, "load restored snapshot", err)
	if loadedSnap.Version.Int() != 1 {
		t.Errorf("restored snapshot version = %d, want 1", loadedSnap.Version.Int())
	}

	loadedCP, err := restored.CheckpointStore().Load(ctx, "users-projection")
	must(t, "load restored checkpoint", err)
	if loadedCP.EventID != cpEventID {
		t.Errorf(
			"restored checkpoint EventID = %s, want %s (pre-backup)",
			loadedCP.EventID,
			cpEventID,
		)
	}

	evtA3, _ := event.NewEvent("UserDeleted", streamA, "User", 2, []byte(`{}`))
	must(
		t,
		"save to restored backend",
		restored.EventStore().Save(ctx, refA, []event.Event{evtA3}, 1),
	)

	loadedA2, err := restored.EventStore().Load(ctx, refA)
	must(t, "load after new write", err)
	if len(loadedA2) != 2 {
		t.Errorf("after new write: %d events, want 2", len(loadedA2))
	}
}

// RunIncrementalCheckpoints verifies that multiple sequential backups capture
// the state at each point in time.
func RunIncrementalCheckpoints(t *testing.T, f Factory) {
	t.Parallel()

	ctx := context.Background()
	source := f.New(t)
	t.Cleanup(func() { _ = source.Close() })

	store := source.EventStore()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	evt1, _ := event.NewEvent("Incremented", streamID, "Counter", 1, []byte(`{}`))
	must(t, "save evt1", store.Save(ctx, ref, []event.Event{evt1}, 0))

	backup1 := filepath.Join(t.TempDir(), "b1")
	f.Backup(t, source, backup1)

	evt2, _ := event.NewEvent("Incremented", streamID, "Counter", 2, []byte(`{}`))
	must(t, "save evt2", store.Save(ctx, ref, []event.Event{evt2}, 1))

	backup2 := filepath.Join(t.TempDir(), "b2")
	f.Backup(t, source, backup2)

	r1 := f.Restore(t, backup1)
	t.Cleanup(func() { _ = r1.Close() })

	loaded1, err := r1.EventStore().Load(ctx, ref)
	must(t, "load backup1", err)
	if len(loaded1) != 1 {
		t.Errorf("backup1: %d events, want 1", len(loaded1))
	}

	r2 := f.Restore(t, backup2)
	t.Cleanup(func() { _ = r2.Close() })

	loaded2, err := r2.EventStore().Load(ctx, ref)
	must(t, "load backup2", err)
	if len(loaded2) != 2 {
		t.Errorf("backup2: %d events, want 2", len(loaded2))
	}
}

func must(t *testing.T, label string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
}
