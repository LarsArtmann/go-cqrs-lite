package bboltengine_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	bboltengine "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine must clear EVERY ADT surface: all engine data lives in the
// single cqrs_meta bucket, so a bucket drop that leaves anything behind is a
// layout bug, not a coverage gap.
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)
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

// The drop is scoped to the engine bucket: a caller-owned bbolt DB shared
// with foreign buckets must keep them across a reset.
func TestResetEngine_KeepsForeignBuckets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := bolt.Open(filepath.Join(t.TempDir(), "shared.db"), 0o600, nil)
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("foreign_data"))

		return err
	}); err != nil {
		t.Fatalf("plant foreign bucket: %v", err)
	}

	eng, err := bboltengine.NewBboltEngineFromDB(db)
	if err != nil {
		t.Fatalf("NewBboltEngineFromDB: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	if err := mb.MapSet(ctx, "tasks", "t1", "v"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if err := db.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("foreign_data")) == nil {
			return errors.New("foreign bucket must survive the reset")
		}

		if tx.Bucket([]byte("cqrs_meta")) == nil {
			return errors.New("engine bucket must be recreated by the reset")
		}

		return nil
	}); err != nil {
		t.Fatalf("bucket scoping broken: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("engine keys must be gone (ok=%v err=%v)", ok, err)
	}
}

// Sequence numbers stay monotonic across a reset: a consumer holding a
// pre-reset journal token (seq > N) must still see every replayed entry.
func TestResetEngine_SeqMonotonicAcrossReset(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)
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
