package mysqlengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine must clear every ADT surface MySQL implements (map, counter,
// stream log) plus the planned tables; layouts survive so a replay re-fills
// the same tables. Runs only with MYSQL_TEST_DSN set (live server).
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	ctx := context.Background()

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
	if err != nil || len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset (len=%d err=%v)", len(stream), err)
	}
}

// Sequence numbers stay monotonic across a reset: a consumer holding a
// pre-reset journal token (seq > N) must still see every replayed entry.
func TestResetEngine_SeqMonotonicAcrossReset(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
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
