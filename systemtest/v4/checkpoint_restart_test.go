package systemtest_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestEngineCheckpointStoreRestartDurability pins the deployment-declared
// checkpoint contract on a real disk engine: a checkpoint saved on one
// sqlite engine survives close and reopen on the same file — exercising the
// SQL-engine JSON value shape (map[string]any) through reifyCheckpoint.
// Lives in systemtest (not system) because it needs a real engine
// implementation; the store itself is reached through the public
// [system.NewEngineCheckpointStore].
func TestEngineCheckpointStoreRestartDurability(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "checkpoints.db")

	first, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	if err != nil {
		t.Fatalf("open first engine: %v", err)
	}

	want := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now().UTC()}

	store := system.NewEngineCheckpointStore(first.(metaengine.MapBackend))
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

	reopened := system.NewEngineCheckpointStore(second.(metaengine.MapBackend))
	got, err := reopened.Load(ctx, "orders-projection")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}

	if got.EventID != want.EventID || !got.ProcessedAt.Equal(want.ProcessedAt) {
		t.Fatalf("checkpoint lost across restart: got (%s, %s), want (%s, %s)",
			got.EventID, got.ProcessedAt, want.EventID, want.ProcessedAt)
	}
}
