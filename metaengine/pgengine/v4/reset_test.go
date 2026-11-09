package pgengine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type resetTask struct {
	ID     string
	Status string
}

func resetTaskQuery() metaengine.QueryDecl[resetTask, resetTask] {
	return metaengine.Query[resetTask, resetTask](
		"pg_reset_tasks",
		metaengine.OnRecordTyped(
			"task_created",
			resetTask{},
			func(_ record.Record, e resetTask) (string, resetTask) { return e.ID, e },
		),
	)
}

// ResetEngine must clear every engine-owned table and keep layouts: after a
// reset, reads see an empty engine and Store.Reset reports postgres cleared
// (non-partial one-call revert).
func TestResetEngine_ClearsAllState(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)
	ctx := context.Background()

	mb := eng.(metaengine.MapBackend)
	cb := eng.(metaengine.CounterBackend)
	sl := eng.(metaengine.StreamLogBackend)

	if err := mb.MapSet(ctx, "tasks", "t1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"created": 2}); err != nil {
		t.Fatalf("CounterIncrement: %v", err)
	}

	// Run-unique journal key: the journal survives resets (ADR-0143) and a
	// reused test database accumulates journal rows across runs — the
	// survival assertion stays meaningful on long-lived servers.
	streamKey := fmt.Sprintf("s1-%d", time.Now().UnixNano())

	if err := sl.StreamAppend(ctx, "events", streamKey, []any{"e1"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	// ADR-0142 write-side collections ride the same reset contract.
	claimer := eng.(metaengine.DueClaimer)
	if err := claimer.ClaimInsert(
		ctx,
		"timers",
		"t1",
		time.Now().Add(-time.Second),
		[]byte("fire"),
	); err != nil {
		t.Fatalf("ClaimInsert: %v", err)
	}

	dedup := eng.(metaengine.DedupStore)
	if seen, err := dedup.DedupCheckAndRecord(
		ctx,
		"cmds",
		"c1",
		time.Minute,
		time.Now(),
	); err != nil ||
		seen {
		t.Fatalf("DedupCheckAndRecord first (seen=%v err=%v)", seen, err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("map must be empty after reset (ok=%v err=%v)", ok, err)
	}

	counters, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	if len(counters) != 0 {
		t.Fatalf("counters must be empty after reset, got %v", counters)
	}

	stream, err := sl.StreamRead(ctx, "events", streamKey)
	if err != nil || len(stream) != 1 {
		t.Fatalf(
			"stream log (journal, facts) must SURVIVE reset (ADR-0143): got %d entries",
			len(stream),
		)
	}

	claims, err := claimer.ClaimDue(ctx, metaengine.ClaimDueRequest{
		Collection: "timers", Owner: "w1", Lease: time.Minute,
	})
	if err != nil || len(claims) != 0 {
		t.Fatalf("claims must be gone after reset (len=%d err=%v)", len(claims), err)
	}

	seen, err := dedup.DedupSeen(ctx, "cmds", "c1", time.Now())
	if err != nil || seen {
		t.Fatalf("dedup must be gone after reset (seen=%v err=%v)", seen, err)
	}
}

func TestStore_Reset_ClearsPostgresEngine(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, resetTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	result, err := store.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if result.Partial() {
		t.Fatalf("postgres implements EngineResetter, reset must not be partial: %s", result)
	}

	if len(result.ClearedEngines) != 1 || result.ClearedEngines[0] != "postgres" {
		t.Fatalf("expected postgres in ClearedEngines, got %v", result.ClearedEngines)
	}
}
