package system

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestCheckpointEngineResolution pins the deployment-declared checkpoint
// engine: the engine named "checkpoints" wins, else the first engine, else
// nil — and only engines carrying the Map ADT qualify.
func TestCheckpointEngineResolution(t *testing.T) {
	t.Parallel()

	primary := metaengine.NewMemoryEngine()
	dedicated := metaengine.NewMemoryEngine()
	dedicatedBackend := dedicated.(metaengine.MapBackend)
	primaryBackend := primary.(metaengine.MapBackend)

	named := &System{engines: []namedEngine{
		{name: "primary", engine: primary},
		{name: "checkpoints", engine: dedicated},
	}}
	if got := named.checkpointEngine(); got == nil || got != dedicatedBackend {
		t.Fatal("the engine named 'checkpoints' must win checkpoint storage")
	}

	fallback := &System{engines: []namedEngine{{name: "primary", engine: primary}}}
	if got := fallback.checkpointEngine(); got == nil || got != primaryBackend {
		t.Fatal("checkpoint storage must fall back to the first engine")
	}

	if got := (&System{}).checkpointEngine(); got != nil {
		t.Fatal("no engines must resolve to a nil checkpoint engine")
	}
}

// TestEngineCheckpointStoreRoundTrip covers the memory-engine value shape
// (typed struct in, typed struct out) and the missing-key contract.
func TestEngineCheckpointStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	backend := metaengine.NewMemoryEngine()
	store := &engineCheckpointStore{engine: backend.(metaengine.MapBackend)}

	if cp, err := store.Load(ctx, "never-saved"); err != nil || !cp.IsZero() {
		t.Fatalf("missing checkpoint = (%v, %v), want zero, nil", cp, err)
	}

	want := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now().UTC()}
	if err := store.Save(ctx, "orders-projection", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx, "orders-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.EventID != want.EventID || !got.ProcessedAt.Equal(want.ProcessedAt) {
		t.Fatalf("round-trip mismatch: got (%s, %s), want (%s, %s)",
			got.EventID, got.ProcessedAt, want.EventID, want.ProcessedAt)
	}
}

// TestEngineCheckpointStoreRestartDurability is the T12 restart gate: a
// checkpoint saved on one sqlite engine survives close and reopen on the same
// file — exercising the SQL-engine JSON value shape (map[string]any) through
// reifyCheckpoint.
func TestEngineCheckpointStoreRestartDurability(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "checkpoints.db")

	first, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	if err != nil {
		t.Fatalf("open first engine: %v", err)
	}

	want := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now().UTC()}

	store := &engineCheckpointStore{engine: first.(metaengine.MapBackend)}
	if err := store.Save(ctx, "orders-projection", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close first engine: %v", err)
	}

	second, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	if err != nil {
		t.Fatalf("reopen engine: %v", err)
	}

	defer func() { _ = second.Close() }()

	reopened := &engineCheckpointStore{engine: second.(metaengine.MapBackend)}
	got, err := reopened.Load(ctx, "orders-projection")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}

	if got.EventID != want.EventID || !got.ProcessedAt.Equal(want.ProcessedAt) {
		t.Fatalf("checkpoint lost across restart: got (%s, %s), want (%s, %s)",
			got.EventID, got.ProcessedAt, want.EventID, want.ProcessedAt)
	}
}
