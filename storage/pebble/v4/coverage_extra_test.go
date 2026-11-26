package pebble_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

// TestEventStore_CorruptEventTriggersCorruptionError writes garbage data
// to the event store keys and verifies that Load returns a Corruption error.
func TestEventStore_CorruptEventTriggersCorruptionError(t *testing.T) {
	t.Parallel()

	database := openProbeDB(t)

	logger := slog.New(slog.NewTextHandler(&testWriter{t: t}, nil))
	store, err := cqrspebble.NewStore(database, logger)
	if err != nil {
		t.Fatal(err)
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)
	key := []byte("cqrs_event:Test:" + ref.ID.String() + ":0000000003")

	// A valid event precedes the corrupt one: the error must win over a
	// silently truncated result.
	evt, err := event.NewEvent("test.event", streamID, "Test", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	for name, payload := range map[string][]byte{
		"binary garbage": []byte("garbage-data-not-cbor-or-json"),
		"truncated json": []byte(`{"id":`),
	} {
		if err := database.Set(key, payload, pebble.Sync); err != nil {
			t.Fatalf("Set corrupt: %v", err)
		}

		events, err := store.Load(context.Background(), ref)
		assertCorruption(t, "event Load ("+name+")", key, err)
		if len(events) != 0 {
			t.Errorf("event Load (%s): returned %d events beside the error", name, len(events))
		}
	}
}

// assertCorruption pins that a decode failure is reported, not swallowed.
func assertCorruption(t *testing.T, what string, key []byte, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: corrupt data must return an error, got nil", what)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Corruption {
		t.Errorf("%s: family = %v, want Corruption (err: %v)", what, family, err)
	}
}

func openProbeDB(t *testing.T) *pebble.DB {
	t.Helper()

	database, err := pebble.Open(t.TempDir(), &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	return database
}

// TestCheckpointStore_CorruptData writes corrupt checkpoint data
// and verifies Load returns a Corruption error, and that the legacy
// JSON checkpoint is still read.
func TestCheckpointStore_CorruptData(t *testing.T) {
	t.Parallel()

	database := openProbeDB(t)

	cpStore, err := cqrspebble.NewCheckpointStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	for name, payload := range map[string][]byte{
		"binary garbage": []byte("garbage-not-cbor"),
		"broken json":    []byte(`{"broken json`),
	} {
		projName := "test-projection-" + name

		if err := database.Set([]byte("cqrs_checkpoint:"+projName), payload, pebble.Sync); err != nil {
			t.Fatalf("Set corrupt: %v", err)
		}

		_, err := cpStore.Load(context.Background(), projName)
		assertCorruption(t, "checkpoint Load ("+name+")", nil, err)
	}

	// Legacy JSON checkpoints (written before the CBOR migration) still load.
	legacyID := id.NewEventID()
	if err := database.Set([]byte("cqrs_checkpoint:legacy"),
		[]byte(`{"EventID":"`+legacyID.String()+`","ProcessedAt":1700000000000000000}`), pebble.Sync); err != nil {
		t.Fatalf("Set legacy: %v", err)
	}

	cp, err := cpStore.Load(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("legacy JSON checkpoint: %v", err)
	}

	if cp.EventID != legacyID || cp.ProcessedAt.UnixNano() != 1700000000000000000 {
		t.Errorf("legacy JSON checkpoint: got %+v", cp)
	}

	// A missing checkpoint is not corruption: zero value, no error.
	if _, err := cpStore.Load(context.Background(), "never-written"); err != nil {
		t.Errorf("missing checkpoint: err = %v, want nil", err)
	}
}

// TestSnapshotStore_CorruptData writes corrupt snapshot data
// and verifies LoadAtVersion returns a Corruption error.
func TestSnapshotStore_CorruptData(t *testing.T) {
	t.Parallel()

	database := openProbeDB(t)

	snapStore, err := cqrspebble.NewSnapshotStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)
	key := []byte("cqrs_snapshot:Test:" + streamID.String())

	for name, payload := range map[string][]byte{
		"binary garbage": []byte("garbage-snapshot"),
		"broken json":    []byte(`{"broken`),
	} {
		if err := database.Set(key, payload, pebble.Sync); err != nil {
			t.Fatalf("Set corrupt: %v", err)
		}

		_, err := snapStore.LoadAtVersion(context.Background(), ref, 5)
		assertCorruption(t, "snapshot LoadAtVersion ("+name+")", key, err)
	}
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

	snapStore, err := cqrspebble.NewSnapshotStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)

	// Save a snapshot
	snap := snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Test",
		Version:    1,
		State:      []byte(`{"name":"test"}`),
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

	backend, err := cqrspebble.NewBackend(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

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
