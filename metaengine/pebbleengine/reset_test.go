package pebbleengine_test

import (
	"context"
	"testing"

	"github.com/cockroachdb/pebble"

	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine must clear EVERY ADT surface: a reset that misses one key
// range leaves stale state a replay folds on top of.
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	ctx := context.Background()
	resetter := eng.(metaengine.EngineResetter)

	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.SetBackend)
	cb := eng.(metaengine.CounterBackend)
	mm := eng.(metaengine.MultimapBackend)
	lb := eng.(metaengine.LogBackend)
	sl := eng.(metaengine.StreamLogBackend)

	if err := mb.MapSet(ctx, "tasks", "t1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := sb.SetAdd(ctx, "seen", "t1"); err != nil {
		t.Fatalf("SetAdd: %v", err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"created": 3}); err != nil {
		t.Fatalf("CounterIncrement: %v", err)
	}

	if err := mm.MultiAdd(ctx, "index", "t1", "v1"); err != nil {
		t.Fatalf("MultiAdd: %v", err)
	}

	if err := lb.LogAppend(ctx, "audit", "entry"); err != nil {
		t.Fatalf("LogAppend: %v", err)
	}

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1", "e2"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("map must be empty after reset (ok=%v err=%v)", ok, err)
	}

	if ok, err := sb.SetContains(ctx, "seen", "t1"); err != nil || ok {
		t.Fatalf("set must be empty after reset (ok=%v err=%v)", ok, err)
	}

	counters, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	if len(counters) != 0 {
		t.Fatalf("counters must be empty after reset, got %v", counters)
	}

	values, err := mm.MultiGet(ctx, "index", "t1")
	if err != nil || len(values) != 0 {
		t.Fatalf("multimap must be empty after reset (len=%d err=%v)", len(values), err)
	}

	entries, err := lb.LogTail(ctx, "audit", 10)
	if err != nil || len(entries) != 0 {
		t.Fatalf("log must be empty after reset (len=%d err=%v)", len(entries), err)
	}

	stream, err := sl.StreamRead(ctx, "events", "s1")
	if err != nil || len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset (len=%d err=%v)", len(stream), err)
	}
}

// Layout declarations survive a reset (post-Plan state), and the replay's
// writes rebuild the secondary indexes from zero.
func TestResetEngine_LayoutSurvivesAndRebuilds(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
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

	sc := eng.(metaengine.ScanBackend)
	res, err := sc.MapScan(ctx, "tasks",
		func(item any) bool { m, _ := item.(map[string]any); return m["status"] == "done" },
		nil, nil, 10,
	)
	if err != nil {
		t.Fatalf("MapScan after reset: %v", err)
	}

	if len(res.Items) != 1 {
		t.Fatalf("post-reset scan must see exactly the replayed row, got %d", len(res.Items))
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("pre-reset rows must be gone (ok=%v err=%v)", ok, err)
	}
}

// The delete is scoped to engine-owned prefixes: a caller-owned pebble.DB
// shared with foreign keys must keep them across a reset.
func TestResetEngine_KeepsForeignKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := pebble.Open("", &pebble.Options{})
	if err != nil {
		t.Skipf("pebble not available: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := db.Set([]byte("foreign\x00keep"), []byte("mine"), nil); err != nil {
		t.Fatalf("plant foreign key: %v", err)
	}

	eng, err := pebbleengine.NewPebbleEngineFromDB(db)
	if err != nil {
		t.Fatalf("NewPebbleEngineFromDB: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	if err := mb.MapSet(ctx, "tasks", "t1", "v"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	val, closer, err := db.Get([]byte("foreign\x00keep"))
	if err != nil {
		t.Fatalf("foreign key must survive the reset: %v", err)
	}

	_ = closer.Close()

	if string(val) != "mine" {
		t.Fatalf("foreign key value corrupted: %q", val)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("engine keys must be gone (ok=%v err=%v)", ok, err)
	}
}

// Sequence numbers stay monotonic across a reset: a consumer holding a
// pre-reset journal token (seq > N) must still see every replayed entry.
func TestResetEngine_SeqMonotonicAcrossReset(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
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
		t.Fatalf("journal seq must stay monotonic across reset: before=%d after=%d", lastSeq, after[0].Seq)
	}
}
