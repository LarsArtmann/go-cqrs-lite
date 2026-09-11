package pgengine_test

import (
	"context"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

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

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
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

	stream, err := sl.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset, got %d entries", len(stream))
	}
}

func TestStore_Reset_ClearsPostgresEngine(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, pgFindTaskQuery())
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
