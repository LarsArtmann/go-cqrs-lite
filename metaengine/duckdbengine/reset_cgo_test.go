//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type duckGraphAdder interface {
	GraphAddEdge(ctx context.Context, col string, edge metaengine.Edge) error
}

type duckGraphReader interface {
	GraphNeighbors(ctx context.Context, col string, node any, depth int) ([]any, error)
}

// ResetEngine must clear every ADT surface DuckDB implements (map, counter,
// stream log, graph) plus the planned tables; layouts survive so a replay
// re-fills the same tables.
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	ctx := context.Background()
	resetter := eng.(metaengine.EngineResetter)

	mb := eng.(metaengine.MapBackend)
	cb := eng.(metaengine.CounterBackend)
	sl := eng.(metaengine.StreamLogBackend)

	if err := mb.MapSet(ctx, "tasks", "t1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"created": 3}); err != nil {
		t.Fatalf("CounterIncrement: %v", err)
	}

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1", "e2"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	if err := eng.(duckGraphAdder).GraphAddEdge(
		ctx,
		"graph",
		metaengine.Edge{From: "a", To: "b"},
	); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
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
	if err != nil || len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset (len=%d err=%v)", len(stream), err)
	}

	neighbors, err := eng.(duckGraphReader).GraphNeighbors(ctx, "graph", "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	if len(neighbors) != 0 {
		t.Fatalf("graph must be empty after reset, got %v", neighbors)
	}
}

// Layout declarations survive a reset; post-reset writes land in the planned
// table again (meta_map stays untouched).
func TestResetEngine_KeepsPlannedLayout(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	ctx := context.Background()

	planner := eng.(metaengine.LayoutPlanner)
	if err := planner.ApplyLayout("tasks", []string{"status"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	if err := mb.MapSet(ctx, "tasks", "t1", map[string]any{"status": "open"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if err := mb.MapSet(ctx, "tasks", "t2", map[string]any{"status": "done"}); err != nil {
		t.Fatalf("MapSet after reset: %v", err)
	}

	val, ok, err := mb.MapGet(ctx, "tasks", "t2")
	if err != nil || !ok {
		t.Fatalf("post-reset planned write must be readable (ok=%v err=%v)", ok, err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("pre-reset rows must be gone (ok=%v err=%v)", ok, err)
	}

	_ = val
}

// Sequence numbers stay monotonic across a reset: a consumer holding a
// pre-reset journal token (seq > N) must still see every replayed entry.
func TestResetEngine_SeqMonotonicAcrossReset(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	ctx := context.Background()
	sl := eng.(metaengine.StreamLogBackend)
	seqLog := eng.(metaengine.SeqSeekableStreamLog)

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	before, err := seqLog.JournalReadAllWithSeq(ctx, "events")
	if err != nil || len(before) != 1 {
		t.Fatalf("JournalReadAllWithSeq before reset (len=%d err=%v)", len(before), err)
	}

	lastSeq := before[len(before)-1].Seq

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"replayed"}); err != nil {
		t.Fatalf("StreamAppend after reset: %v", err)
	}

	after, err := seqLog.JournalReadAllWithSeq(ctx, "events")
	if err != nil {
		t.Fatalf("JournalReadAllWithSeq after reset: %v", err)
	}

	if len(after) != 1 {
		t.Fatalf("exactly the replayed entry must exist after reset, got %d", len(after))
	}

	if after[0].Seq <= lastSeq {
		t.Fatalf(
			"journal seq must stay monotonic across reset: before=%d after=%d",
			lastSeq,
			after[0].Seq,
		)
	}
}
