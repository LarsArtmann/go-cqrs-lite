package system_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_SnapshotAdapterDirect verifies the SnapshotAdapter Save/Load/Delete
// roundtrip through the System with a SQLite engine (which implements
// SnapshotBackend).
func TestSystem_SnapshotAdapterDirect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	snapStore := sys.SnapshotStore()
	if snapStore == nil {
		t.Fatal("expected non-nil snapshot store — SQLite engine implements SnapshotBackend")
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)

	// Save a snapshot.
	state := TaskState{Title: "test-snap", Status: "pending", Exists: true}
	stateBytes, _ := json.Marshal(state)

	snap := snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Task",
		Version:    event.Version(5),
		State:      stateBytes,
		CreatedAt:  time.Now(),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load it back.
	loaded, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded == nil {
		t.Fatal("expected non-nil snapshot after load")
	}

	if loaded.Version != event.Version(5) {
		t.Fatalf("expected version 5, got %d", loaded.Version)
	}

	var loadedState TaskState
	if err := json.Unmarshal(loaded.State, &loadedState); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	if loadedState.Title != "test-snap" {
		t.Fatalf("expected title 'test-snap', got %q", loadedState.Title)
	}

	// Delete it.
	if err := snapStore.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone.
	again, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}

	if again != nil {
		t.Fatal("expected nil snapshot after delete")
	}
}

// TestSystem_SnapshotAdapterLoadAtVersion verifies version-bounded snapshot loading.
func TestSystem_SnapshotAdapterLoadAtVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	snapStore := sys.SnapshotStore()
	if snapStore == nil {
		t.Fatal("expected non-nil snapshot store")
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)
	stateBytes, _ := json.Marshal(TaskState{Title: "v10", Status: "pending"})

	// Save at version 10.
	if err := snapStore.Save(ctx, snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Task",
		Version:    event.Version(10),
		State:      stateBytes,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// LoadAtVersion(15) should find it (latest at or below 15).
	snap, err := snapStore.LoadAtVersion(ctx, ref, event.Version(15))
	if err != nil {
		t.Fatalf("LoadAtVersion 15: %v", err)
	}

	if snap == nil {
		t.Fatal("expected snapshot at version <= 15")
	}

	if snap.Version != event.Version(10) {
		t.Fatalf("expected version 10, got %d", snap.Version)
	}

	// LoadAtVersion(5) should NOT find it (snapshot is at 10 > 5).
	none, err := snapStore.LoadAtVersion(ctx, ref, event.Version(5))
	if err != nil {
		t.Fatalf("LoadAtVersion 5: %v", err)
	}

	if none != nil {
		t.Fatal("expected nil snapshot for version 5 when snapshot is at 10")
	}
}

// TestSystem_SnapshotE2E_DeciderLifecycle verifies the full decider → snapshot → reload
// cycle through the System with a SQLite engine. The decider is registered with
// WithSnapshotStrategy so snapshots fire automatically on writes.
func TestSystem_SnapshotE2E_DeciderLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			// Register decider with snapshot strategy: snapshot after every 1 event.
			strategy, err2 := snapshot.EveryNEvents(1)
			if err2 != nil {
				panic(err2)
			}

			if err := system.RegisterDecider(sys, "Task", TaskDecider,
				system.WithSnapshotStrategy(strategy),
			); err != nil {
				panic(err)
			}

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "snap-test", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.complete",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.completed",
								cmd.StreamID(), "Task", ver+1,
								TaskCompleted{At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	snapStore := sys.SnapshotStore()
	if snapStore == nil {
		t.Fatal("expected non-nil snapshot store — SQLite engine implements SnapshotBackend")
	}

	// Dispatch a command to create the task.
	streamID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID)); err != nil {
		t.Fatalf("dispatch task.create: %v", err)
	}

	// Verify a snapshot was saved.
	ref := id.NewStreamRef("Task", streamID)
	snap, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load snapshot after create: %v", err)
	}

	if snap == nil {
		t.Fatal("expected snapshot to be saved after task.create (EveryNEvents(1))")
	}

	if snap.Version != event.Version(1) {
		t.Fatalf("expected snapshot version 1, got %d", snap.Version)
	}

	// Verify the snapshot state is correct.
	var snapState TaskState
	if err := json.Unmarshal(snap.State, &snapState); err != nil {
		t.Fatalf("unmarshal snapshot state: %v", err)
	}

	if snapState.Title != "snap-test" {
		t.Fatalf("expected snapshot title 'snap-test', got %q", snapState.Title)
	}

	if snapState.Status != "pending" {
		t.Fatalf("expected snapshot status 'pending', got %q", snapState.Status)
	}

	// Dispatch another command to advance the stream.
	if err := sys.CommandDispatcher().Dispatch(ctx, newCmd("task.complete", streamID)); err != nil {
		t.Fatalf("dispatch task.complete: %v", err)
	}

	// Verify the snapshot was updated.
	snap2, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load snapshot after complete: %v", err)
	}

	if snap2 == nil {
		t.Fatal("expected snapshot to exist after task.complete")
	}

	if snap2.Version != event.Version(2) {
		t.Fatalf("expected snapshot version 2, got %d", snap2.Version)
	}

	var snap2State TaskState
	if err := json.Unmarshal(snap2.State, &snap2State); err != nil {
		t.Fatalf("unmarshal snap2 state: %v", err)
	}

	if snap2State.Status != "completed" {
		t.Fatalf("expected status 'completed', got %q", snap2State.Status)
	}
}
