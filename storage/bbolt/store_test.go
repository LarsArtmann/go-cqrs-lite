package bbolt

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	evt, err := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	err = store.Save(ctx, ref, []event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "user.created" {
		t.Fatalf("expected type user.created, got %s", loaded[0].Type())
	}
}

func TestVersionConflict(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	evt, _ := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{"name":"alice"}`))
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("save first: %v", err)
	}

	evt2, _ := event.NewEvent("user.updated", ref.ID, "User", 2, []byte(`{"name":"bob"}`))
	err := store.Save(ctx, ref, []event.Event{evt2}, 0)
	if err == nil {
		t.Fatal("expected version conflict error, got nil")
	}
}

func TestJournalReadAll(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		ref := id.NewStreamRef("User", id.NewStreamID())
		evt, _ := event.NewEvent("user.created", ref.ID, "User", 1,
			[]byte(`{"i":`+string(rune('0'+i))+`}`))
		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	events, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events in journal, got %d", len(events))
	}
}

func TestCheckpointSaveLoad(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cpStore := backend.CheckpointStore()
	ctx := context.Background()

	cp := event.Checkpoint{
		EventID: id.NewEventID(),
	}

	if err := cpStore.Save(ctx, "test-projection", cp); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	loaded, err := cpStore.Load(ctx, "test-projection")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}

	if loaded.EventID != cp.EventID {
		t.Fatal("checkpoint event ID mismatch")
	}
}

func TestSnapshotSaveLoad(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	snapStore := backend.SnapshotStore()
	ctx := context.Background()

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("User", streamID)
	snap := snapshot.Snapshot{
		StreamType: "User",
		StreamID:   streamID,
		Version:    5,
		State:      []byte(`{"name":"alice"}`),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	loaded, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	if loaded.Version != 5 {
		t.Fatalf("expected version 5, got %d", loaded.Version)
	}

	if string(loaded.State) != `{"name":"alice"}` {
		t.Fatalf("unexpected state: %s", loaded.State)
	}
}

func TestKVAdapterSetGet(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	kv := backend.ReadModels()
	ctx := context.Background()

	if err := kv.Set(ctx, []byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := kv.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if string(val) != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}

	if _, err := kv.Get(ctx, []byte("missing")); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func newTestBackend(t *testing.T) *Backend {
	t.Helper()

	dir := t.TempDir()
	backend, err := Open(filepath.Join(dir, "test.db"), nil)
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	return backend
}
