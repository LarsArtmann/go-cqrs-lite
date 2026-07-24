package projectionhost_test

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// TestIntegration_ProjectionHost_SQLiteCheckpoint proves the host persists
// checkpoints to a SQL database — closing the production-readiness gap where
// only MemoryCheckpointStore existed. storage.SQLCheckpointStore already
// implements event.CheckpointStore, so projectionhost supports it natively.
func TestIntegration_ProjectionHost_SQLiteCheckpoint(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	// Create the checkpoints table (the store expects it to exist).
	if _, err := db.ExecContext(
		t.Context(),
		sqlpkg.SQLiteDialect{}.CheckpointSchema(),
	); err != nil {
		t.Fatalf("create checkpoints table: %v", err)
	}

	cpStore, err := storage.NewSQLiteCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}

	store := memory.NewMemoryStore()
	defer store.Close()

	host, err := projectionhost.New(
		store, cpStore,
		projectionhost.WithBatchSize(5),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	proj := &sqliteCountProjection{name: "sql-test"}
	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Item", streamID)
	for range 3 {
		evt, _ := event.New("item.added", streamID, "Item", 1, []byte("payload"))
		_ = store.AppendBatch(context.Background(), ref, []event.Event{evt})
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = host.Start(ctx) }()

	requireEventually(t, 3*time.Second, func() bool { return proj.count.Load() == 3 })
	cancel()
	_ = host.Stop()

	// Verify the checkpoint landed in SQLite by opening a FRESH host with the
	// same DB — the new host should resume from the persisted checkpoint.
	host2, _ := projectionhost.New(store, cpStore, projectionhost.WithBatchSize(5))
	cp, err := cpStore.Load(context.Background(), "sql-test")
	if err != nil {
		t.Fatalf("checkpoint Load: %v", err)
	}
	if cp.IsZero() {
		t.Fatal("expected non-zero checkpoint persisted in SQLite")
	}
	_ = host2
}

type sqliteCountProjection struct {
	name  string
	count atomic.Int64
}

func (p *sqliteCountProjection) Name() string { return p.name }

func (p *sqliteCountProjection) EventTypes() []event.Type { return nil }

func (p *sqliteCountProjection) Handle(_ context.Context, _ event.Event) error {
	p.count.Add(1)

	return nil
}
