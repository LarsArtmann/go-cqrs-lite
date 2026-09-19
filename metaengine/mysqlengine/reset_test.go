package mysqlengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine must clear every ADT surface MySQL implements (map, counter,
// stream log) plus the planned tables; layouts survive so a replay re-fills
// the same tables. Runs only with MYSQL_TEST_DSN set (live server).
//
// NOT parallel: the engine shares the persistent cqrs_test database with
// every other live test, and ResetEngine is a TOTAL wipe — a reset racing a
// parallel reader silently deletes its data mid-test (2026-09-16: layout,
// pushdown and stream-log tests failed nondeterministically inside the
// reset window). Non-parallel tests never overlap any other test, which is
// the exclusivity mechanism (no locks needed).
func TestResetEngine_ClearsEveryADT(t *testing.T) {
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

	stream, err := sl.StreamRead(ctx, "events", "s1")
	if err != nil || len(stream) != 2 {
		t.Fatalf("stream log (journal, facts) must SURVIVE reset (ADR-0143): len=%d err=%v", len(stream), err)
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

// Sequence numbers stay monotonic across a reset: a consumer holding a
// pre-reset journal token (seq > N) must still see every replayed entry.
// NOT parallel: see TestResetEngine_ClearsEveryADT.
func TestResetEngine_SeqMonotonicAcrossReset(t *testing.T) {
	eng := mustNewMySQLEngine(t)
	//art-dupl:accept intentional cross-module mirror — each dep-isolated engine module carries its own reset test/body (ADR-0136); see AGENTS.md #19
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

	if len(after) != 2 {
		t.Fatalf("journal (facts) must SURVIVE reset (ADR-0143): want the pre-reset entry plus the appended one, got %d", len(after))
	}

	if after[len(after)-1].Seq <= lastSeq {
		t.Fatalf(
			"journal seq must stay monotonic across reset: before=%d after=%d",
			lastSeq,
			after[len(after)-1].Seq,
		)
	}
}
