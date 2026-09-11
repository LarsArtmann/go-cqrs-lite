package dgraphengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type dgraphGraphAdder interface {
	GraphAddEdge(ctx context.Context, col string, edge metaengine.Edge) error
}

type dgraphGraphReader interface {
	GraphNeighbors(ctx context.Context, col string, node any, depth int) ([]any, error)
}

// ResetEngine must clear EVERY ADT surface: the upsert deletes all nodes
// carrying an engine dgraph.type, so any ADT left holding data is a
// missing-type bug, not a coverage gap. Live test — skips without a server.
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	col := uniqueCollection(t, "reset")

	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.SetBackend)
	cb := eng.(metaengine.CounterBackend)
	mm := eng.(metaengine.MultimapBackend)
	lb := eng.(metaengine.LogBackend)
	sl := eng.(metaengine.StreamLogBackend)

	if err := mb.MapSet(ctx, col+"_map", "t1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := sb.SetAdd(ctx, col+"_set", "t1"); err != nil {
		t.Fatalf("SetAdd: %v", err)
	}

	if err := cb.CounterIncrement(ctx, col+"_cnt", metaengine.Delta{"created": 3}); err != nil {
		t.Fatalf("CounterIncrement: %v", err)
	}

	if err := mm.MultiAdd(ctx, col+"_mm", "t1", "v1"); err != nil {
		t.Fatalf("MultiAdd: %v", err)
	}

	if err := lb.LogAppend(ctx, col+"_log", "entry"); err != nil {
		t.Fatalf("LogAppend: %v", err)
	}

	if err := sl.StreamAppend(ctx, col+"_sl", "s1", []any{"e1", "e2"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	if err := eng.(dgraphGraphAdder).GraphAddEdge(
		ctx,
		col+"_graph",
		metaengine.Edge{From: "a", To: "b"},
	); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, col+"_map", "t1"); err != nil || ok {
		t.Fatalf("map must be empty after reset (ok=%v err=%v)", ok, err)
	}

	if ok, err := sb.SetContains(ctx, col+"_set", "t1"); err != nil || ok {
		t.Fatalf("set must be empty after reset (ok=%v err=%v)", ok, err)
	}

	counters, err := cb.CounterGet(ctx, col+"_cnt")
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	if len(counters) != 0 {
		t.Fatalf("counters must be empty after reset, got %v", counters)
	}

	values, err := mm.MultiGet(ctx, col+"_mm", "t1")
	if err != nil || len(values) != 0 {
		t.Fatalf("multimap must be empty after reset (len=%d err=%v)", len(values), err)
	}

	entries, err := lb.LogTail(ctx, col+"_log", 10)
	if err != nil || len(entries) != 0 {
		t.Fatalf("log must be empty after reset (len=%d err=%v)", len(entries), err)
	}

	stream, err := sl.StreamRead(ctx, col+"_sl", "s1")
	if err != nil || len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset (len=%d err=%v)", len(stream), err)
	}

	neighbors, err := eng.(dgraphGraphReader).GraphNeighbors(ctx, col+"_graph", "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	if len(neighbors) != 0 {
		t.Fatalf("graph must be empty after reset, got %v", neighbors)
	}
}

// The reset is idempotent (safe to rerun after a partial-failure retry) and
// sequence monotonicity holds by construction — dgraph journal seqs are
// UnixNano timestamps, so post-reset entries always sort after pre-reset ones
// without any counter state to preserve (see reset.go). Live test.
func TestResetEngine_Idempotent(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	col := uniqueCollection(t, "resetidem")

	mb := eng.(metaengine.MapBackend)
	if err := mb.MapSet(ctx, col, "t1", "v1"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	for range 2 {
		if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
			t.Fatalf("ResetEngine (idempotent rerun): %v", err)
		}
	}

	if _, ok, err := mb.MapGet(ctx, col, "t1"); err != nil || ok {
		t.Fatalf("map must be empty after repeated resets (ok=%v err=%v)", ok, err)
	}

	if err := mb.MapSet(ctx, col, "t2", "v2"); err != nil {
		t.Fatalf("MapSet after reset: %v", err)
	}

	val, ok, err := mb.MapGet(ctx, col, "t2")
	if err != nil || !ok {
		t.Fatalf("post-reset write must be readable (ok=%v err=%v)", ok, err)
	}

	if val != "v2" {
		t.Fatalf("post-reset write must round-trip, got %v", val)
	}
}

// ensureEdgeSchema's in-transaction Alter path: creating an edge predicate
// schema from INSIDE RunInTx must succeed (the Alter retries with
// txnScoped=false — schema applies are always retriable). This pins the
// inspection-only claim in transaction.go. Live test.
func TestEnsureEdgeSchema_InsideRunInTx(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	col := uniqueCollection(t, "edgeschema")

	txEng := eng.(interface {
		RunInTx(ctx context.Context, fn func(context.Context) error) error
	})

	if err := txEng.RunInTx(ctx, func(txCtx context.Context) error {
		adder := eng.(dgraphGraphAdder)

		return adder.GraphAddEdge(txCtx, col, metaengine.Edge{From: "a", To: "b"})
	}); err != nil {
		t.Fatalf("GraphAddEdge inside RunInTx (drives ensureEdgeSchema Alter): %v", err)
	}

	neighbors, err := eng.(dgraphGraphReader).GraphNeighbors(ctx, col, "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	if len(neighbors) != 1 {
		t.Fatalf("edge must exist after the in-tx schema ensure, got %v", neighbors)
	}
}
