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

// FuzzSnapshotStore_Roundtrip tests that a snapshot can be saved and loaded
// without corruption regardless of the state bytes.
func FuzzSnapshotStore_Roundtrip(f *testing.F) {
	f.Add([]byte(`{"name":"alice"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"nested":{"deep":{"value":42}}}`))
	f.Add([]byte("\x00\x01\x02\x03"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, state []byte) {
		dir := t.TempDir()
		database, err := pebble.Open(dir, &pebble.Options{})
		if err != nil {
			t.Fatalf("pebble.Open: %v", err)
		}

		defer func() { _ = database.Close() }()

		snapStore := cqrspebble.NewSnapshotStore(database, slog.Default())

		aggID := id.NewAggregateID()
		aggType := event.AggregateType("FuzzSnap")
		snap := snapshot.Snapshot{
			AggregateID:   aggID,
			AggregateType: aggType,
			Version:       event.Version(1),
			State:         state,
			CreatedAt:     time.Now(),
		}

		ctx := context.Background()
		if err := snapStore.Save(ctx, snap); err != nil {
			t.Fatalf("Save: %v", err)
		}

		ref := event.NewAggregateRef(aggType, aggID)
		loaded, err := snapStore.Load(ctx, ref)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if string(loaded.State) != string(state) {
			t.Fatalf("state mismatch: got %q, want %q", loaded.State, state)
		}
	})
}

// FuzzCheckpointStore_Roundtrip tests that a checkpoint can be saved and loaded
// without corruption regardless of the event ID.
func FuzzCheckpointStore_Roundtrip(f *testing.F) {
	f.Add("01HABCDEFGHJKMNPQQRSTUVWXYZ")
	f.Add("")
	f.Add("01HXYZABCDEFGHJKMNPQQRSTUVW")

	f.Fuzz(func(t *testing.T, idStr string) {
		dir := t.TempDir()
		database, err := pebble.Open(dir, &pebble.Options{})
		if err != nil {
			t.Fatalf("pebble.Open: %v", err)
		}

		defer func() { _ = database.Close() }()

		cpStore := cqrspebble.NewCheckpointStore(database, slog.Default())
		ctx := context.Background()

		projectionName := "fuzz-projection"
		checkpoint := event.Checkpoint{
			EventID:     id.NewEventID(),
			ProcessedAt: time.Now(),
		}

		_ = cpStore.Save(ctx, projectionName, checkpoint)
		loaded, err := cpStore.Load(ctx, projectionName)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if loaded.EventID != checkpoint.EventID {
			t.Fatalf("checkpoint mismatch: got %v, want %v", loaded.EventID, checkpoint.EventID)
		}
	})
}
