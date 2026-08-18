package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestCommandAdapter_RoundTrip exercises the CommandAdapter directly over a
// memory engine: batch append, full load, time-filtered loads, and journal
// reads — with deterministic ReceivedAt stamps.
func TestCommandAdapter_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("memory engine must implement StreamLogBackend")
	}

	adapter := system.NewCommandAdapter(backend, "commands")

	ref := command.NewStreamRef("CmdTask", id.NewStreamID())
	early := time.Now().Add(-time.Hour)
	late := time.Now()

	first, err := command.NewPersistedCommand(
		"task.create", ref, []byte(`{"title":"a"}`), command.WithReceivedAt(early),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	second, err := command.NewPersistedCommand(
		"task.update", ref, []byte(`{"title":"b"}`), command.WithReceivedAt(late),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := adapter.Save(ctx, ref, first); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := adapter.AppendBatch(ctx, ref, []*command.PersistedCommand{second}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load = %d commands, want 2", len(loaded))
	}

	mid := early.Add(30 * time.Minute)

	fromTS, err := adapter.LoadFromTimestamp(ctx, ref, mid)
	if err != nil {
		t.Fatalf("LoadFromTimestamp: %v", err)
	}
	if len(fromTS) != 1 || fromTS[0].Type() != "task.update" {
		t.Fatalf("LoadFromTimestamp(mid) = %v, want only task.update", fromTS)
	}

	toTS, err := adapter.LoadToTimestamp(ctx, ref, mid)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}
	if len(toTS) != 1 || toTS[0].Type() != "task.create" {
		t.Fatalf("LoadToTimestamp(mid) = %v, want only task.create", toTS)
	}

	readAll, err := adapter.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(readAll) != 2 {
		t.Fatalf("ReadAll = %d commands, want 2", len(readAll))
	}

	readFrom, err := adapter.ReadFrom(ctx, id.CommandID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if len(readFrom) != 2 {
		t.Fatalf("ReadFrom = %d commands, want 2", len(readFrom))
	}
}
