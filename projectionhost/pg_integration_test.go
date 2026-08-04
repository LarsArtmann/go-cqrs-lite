//go:build integration

package projectionhost_test

// PostgreSQL integration tests for projectionhost. Run with:
//
//	nix run .#integration-pg -- go test ./projectionhost/... -tags=integration
//
// Uses testcontainers (postgres:16-alpine) when DATABASE_URL/
// POSTGRES_TEST_DSN is unset. Each test gets its own fresh database.
//
// These tests verify that the projection host correctly persists and recovers
// checkpoints against a real PostgreSQL instance — the crash-restart scenario
// where a process dies mid-replay and must resume from the last checkpoint.

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// pgDB opens a Postgres connection and applies the CQRS schema (events +
// checkpoints tables). Each test gets its own fresh database.
func pgDB(t *testing.T) *sql.DB {
	t.Helper()

	url := pgTestDSN(t)

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping pg: %v", err)
	}

	if err := storage.PostgresInitSchema(ctx, db); err != nil {
		t.Fatalf("PostgresInitSchema: %v", err)
	}

	return db
}

// restartCountingProjection tracks how many events it has processed. The
// crash-restart test uses this to prove the host does NOT re-process events
// after recovering from a checkpoint.
type restartCountingProjection struct {
	name  string
	count atomic.Int64
}

func (p *restartCountingProjection) Name() string { return p.name }

func (p *restartCountingProjection) EventTypes() []event.Type { return nil }

func (p *restartCountingProjection) Handle(_ context.Context, _ event.Event) error {
	p.count.Add(1)

	return nil
}

// TestIntegration_ProjectionHost_CrashRestart_CheckpointReplay is the M43
// integration test. It verifies that after a simulated crash (host stopped,
// process "restarted"), the projection host loads the persisted checkpoint
// from PostgreSQL and resumes from the correct position — NOT from the
// beginning of the journal.
//
// Scenario:
//  1. Seed 10 events into the event store.
//  2. Run the host until it processes all 10 and persists the checkpoint to PG.
//  3. Stop the host (simulated crash).
//  4. Seed 5 MORE events (the crash happened; new events arrived).
//  5. Start a NEW host with the SAME PG checkpoint store.
//  6. Verify the projection processes only the 5 new events (10+5=15 total),
//     proving the checkpoint was recovered and replay resumed correctly.
func TestIntegration_ProjectionHost_CrashRestart_CheckpointReplay(t *testing.T) {
	t.Parallel()

	db := pgDB(t)

	// PG-backed checkpoint store — survives the simulated crash.
	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		t.Fatalf("NewSQLBackend: %v", err)
	}

	cpStore, err := backend.CheckpointStore()
	if err != nil {
		t.Fatalf("CheckpointStore: %v", err)
	}

	// In-memory event store shared across both host instances. In a real
	// crash-restart, the event store is also persistent (SQL, Pebble); we use
	// memory here because the test isolates the CHECKPOINT recovery path, not
	// event store durability (that's covered by storage/ integration tests).
	eventStore := memory.NewMemoryStore()
	defer eventStore.Close()

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Item", streamID)

	// Phase 1: seed 10 events.
	for range 10 {
		evt, _ := event.New("item.added", streamID, "Item", 1, []byte("phase1"))
		if err := eventStore.AppendBatch(
			context.Background(),
			ref,
			[]event.Event{evt},
		); err != nil {
			t.Fatalf("seed phase1: %v", err)
		}
	}

	// --- First host instance (will "crash"). ---
	proj1 := &restartCountingProjection{name: "crash-test"}

	host1, err := projectionhost.New(
		eventStore, cpStore,
		projectionhost.WithBatchSize(5),
		projectionhost.WithMaxRestarts(-1),
	)
	if err != nil {
		t.Fatalf("New host1: %v", err)
	}

	if err := host1.Register(proj1); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx1, cancel1 := context.WithCancel(context.Background())

	go func() { _ = host1.Start(ctx1) }()

	// Wait for all 10 events to be processed AND the checkpoint persisted.
	requireEventually(t, 5*time.Second, func() bool {
		if proj1.count.Load() != 10 {
			return false
		}

		cp, err := cpStore.Load(context.Background(), "crash-test")

		return err == nil && !cp.IsZero()
	})

	// Simulate crash: stop the host.
	cancel1()
	_ = host1.Stop()

	// Verify checkpoint was persisted to PG.
	cp, err := cpStore.Load(context.Background(), "crash-test")
	if err != nil {
		t.Fatalf("checkpoint Load after crash: %v", err)
	}

	if cp.IsZero() {
		t.Fatal("expected non-zero checkpoint persisted in PostgreSQL after crash")
	}

	// Phase 2: seed 5 MORE events while the host is "down".
	for range 5 {
		evt, _ := event.New("item.added", streamID, "Item", 1, []byte("phase2"))
		if err := eventStore.AppendBatch(
			context.Background(),
			ref,
			[]event.Event{evt},
		); err != nil {
			t.Fatalf("seed phase2: %v", err)
		}
	}

	// --- Second host instance (restart). Uses the SAME PG checkpoint store. ---
	proj2 := &restartCountingProjection{name: "crash-test"}

	host2, err := projectionhost.New(
		eventStore, cpStore,
		projectionhost.WithBatchSize(5),
		projectionhost.WithMaxRestarts(-1),
	)
	if err != nil {
		t.Fatalf("New host2: %v", err)
	}

	if err := host2.Register(proj2); err != nil {
		t.Fatalf("Register host2: %v", err)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	go func() { _ = host2.Start(ctx2) }()

	// The restarted host must process only the 5 NEW events — the checkpoint
	// prevents re-processing the first 10.
	requireEventually(t, 5*time.Second, func() bool {
		return proj2.count.Load() == 5
	})

	cancel2()
	_ = host2.Stop()

	if proj2.count.Load() != 5 {
		t.Fatalf(
			"restarted host should process only 5 new events (checkpoint recovery), got %d",
			proj2.count.Load(),
		)
	}
}
