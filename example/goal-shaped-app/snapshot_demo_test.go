package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestGoal_SnapshotsHappenWithoutDeveloperCode demos the snapshot story:
// "snapshots are a worry the library manages." registerCommands declared one
// strategy line; this test proves the decider then checkpoints long streams
// on its own — no snapshot code anywhere in the domain, and the checkpoint
// replays into the exact state the events fold to.
func TestGoal_SnapshotsHappenWithoutDeveloperCode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := boot(t, ctx, sqliteConfig(t))

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	taskID := id.NewStreamID()

	if err := createTask(ctx, sys, taskID, "Checkpoint me", PriorityNormal); err != nil {
		t.Fatalf("createTask: %v", err)
	}

	if err := dispatch(ctx, sys, cmdCompleteTask, taskID); err != nil {
		t.Fatalf("complete: %v", err)
	}

	snap, err := sys.SnapshotStore().Load(ctx, id.NewStreamRef(streamType, taskID))
	if err != nil {
		t.Fatalf("SnapshotStore.Load: %v", err)
	}

	if snap == nil {
		t.Fatal("expected an automatic snapshot at version 2 (EveryNEvents(2)), got none")
	}

	if snap.Version != event.Version(2) {
		t.Fatalf("snapshot version: got %v, want 2", snap.Version)
	}

	var state TaskState
	if err := json.Unmarshal(snap.State, &state); err != nil {
		t.Fatalf("decode snapshot state: %v", err)
	}

	if !state.Exists || !state.Done {
		t.Fatalf("snapshot state: got %+v, want an existing, done task", state)
	}
}
