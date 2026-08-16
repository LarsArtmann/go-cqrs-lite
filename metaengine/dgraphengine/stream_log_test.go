package dgraphengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	enginetest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestStreamLog_AppendRead verifies StreamAppend + StreamRead round-trip on
// Dgraph: values appended to a stream are read back in order.
func TestStreamLog_AppendRead(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	sl, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("dgraphEngine should implement StreamLogBackend")
	}

	ctx := context.Background()
	col := uniqueCollection(t, "test_stream_append_read")

	if err := sl.StreamAppend(ctx, col, "s1", []any{"a", "b", "c"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	values, err := sl.StreamRead(ctx, col, "s1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(values), values)
	}

	for i, want := range []string{"a", "b", "c"} {
		if got, ok := values[i].(string); !ok || got != want {
			t.Errorf("value[%d] = %v, want %q", i, values[i], want)
		}
	}
}

// TestStreamLog_Version verifies StreamVersion returns the correct count.
func TestStreamLog_Version(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	sl := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := uniqueCollection(t, "test_stream_version")

	if v, err := sl.StreamVersion(ctx, col, "s1"); err != nil || v != 0 {
		t.Fatalf("empty stream version = %d, want 0 (err=%v)", v, err)
	}

	if err := sl.StreamAppend(ctx, col, "s1", []any{"x", "y"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	if v, err := sl.StreamVersion(ctx, col, "s1"); err != nil || v != 2 {
		t.Fatalf("stream version = %d, want 2 (err=%v)", v, err)
	}
}

// TestStreamLog_JournalReadAll verifies cross-stream journal ordering.
func TestStreamLog_JournalReadAll(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	sl := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := uniqueCollection(t, "test_stream_journal_all")

	if err := sl.StreamAppend(ctx, col, "s1", []any{"s1-0"}); err != nil {
		t.Fatalf("StreamAppend s1: %v", err)
	}

	if err := sl.StreamAppend(ctx, col, "s2", []any{"s2-0"}); err != nil {
		t.Fatalf("StreamAppend s2: %v", err)
	}

	if err := sl.StreamAppend(ctx, col, "s1", []any{"s1-1"}); err != nil {
		t.Fatalf("StreamAppend s1 again: %v", err)
	}

	all, err := sl.JournalReadAll(ctx, col)
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 journal entries, got %d: %v", len(all), all)
	}
}

// TestStreamLog_JournalReadFrom verifies position-based resumption:
// afterSeq skips exactly that many leading journal entries (the same
// semantics every other engine provides via dense 1-based seqs).
func TestStreamLog_JournalReadFrom(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	sl := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := uniqueCollection(t, "test_stream_journal_from")

	if err := sl.StreamAppend(ctx, col, "s1", []any{"v0", "v1", "v2"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	all, err := sl.JournalReadAll(ctx, col)
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	// Skip exactly one entry — sparse UnixNano seqs must not leak through.
	fromFirst, err := sl.JournalReadFrom(ctx, col, 1, 0)
	if err != nil {
		t.Fatalf("JournalReadFrom(1, 0): %v", err)
	}

	if len(fromFirst) != 2 {
		t.Errorf("JournalReadFrom(1,0) returned %d entries, want 2 (skip exactly one)",
			len(fromFirst))
	}

	if got, _ := fromFirst[0].(string); got != "v1" {
		t.Errorf("JournalReadFrom(1,0)[0] = %v, want %q (append order)", fromFirst[0], "v1")
	}

	// Limit check: first entry only.
	limited, err := sl.JournalReadFrom(ctx, col, 0, 1)
	if err != nil {
		t.Fatalf("JournalReadFrom(0, 1): %v", err)
	}

	if len(limited) != 1 {
		t.Errorf("JournalReadFrom(0,1) returned %d, expected 1", len(limited))
	}

	if got, _ := limited[0].(string); got != "v0" {
		t.Errorf("JournalReadFrom(0,1)[0] = %v, want %q", limited[0], "v0")
	}

	// Resuming past the end returns empty, not an error.
	past, err := sl.JournalReadFrom(ctx, col, 3, 0)
	if err != nil {
		t.Fatalf("JournalReadFrom(3, 0): %v", err)
	}

	if len(past) != 0 {
		t.Errorf("JournalReadFrom(3,0) returned %d entries, want 0", len(past))
	}
}

// TestStreamLog_HarnessParity runs the shared cross-engine contract suite:
// JournalReadFrom positional semantics must match every other engine.
// Uses a unique collection — the Dgraph server persists across tests, and
// the default "events" collection is also written by the ADT matrix.
func TestStreamLog_HarnessParity(t *testing.T) {
	enginetest.RunStreamLogBackendTestIn(t, mustNewDgraphEngine(t), uniqueCollection(t, "events_parity"))
}

// TestStreamLog_AppendExpected verifies optimistic concurrency control.
func TestStreamLog_AppendExpected(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	aa := eng.(metaengine.AtomicAppender)
	ctx := context.Background()
	col := uniqueCollection(t, "test_stream_append_expected")

	// First append at version 0 should succeed.
	if err := aa.StreamAppendExpected(ctx, col, "s1", 0, []any{"first"}); err != nil {
		t.Fatalf("StreamAppendExpected(0): %v", err)
	}

	// Append at wrong version should fail with conflict.
	err := aa.StreamAppendExpected(ctx, col, "s1", 0, []any{"second"})
	if err == nil {
		t.Error("expected ErrVersionConflict for wrong version, got nil")
	}

	// Append at correct version (1) should succeed.
	if err := aa.StreamAppendExpected(ctx, col, "s1", 1, []any{"second"}); err != nil {
		t.Fatalf("StreamAppendExpected(1): %v", err)
	}

	// Verify both values are present.
	sl := eng.(metaengine.StreamLogBackend)
	values, err := sl.StreamRead(ctx, col, "s1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d: %v", len(values), values)
	}
}

// TestStreamLog_ProfileSupportsADT verifies the engine profile declares
// ADTStreamLog in its Supports map.
func TestStreamLog_ProfileSupportsADT(t *testing.T) {
	eng := mustNewDgraphEngine(t)

	profile := eng.Profile()

	c, ok := profile.Supports[metaengine.ADTStreamLog]
	if !ok {
		t.Fatal("dgraph profile should declare ADTStreamLog in Supports")
	}

	if c != metaengine.ComplexityOLogN {
		t.Errorf("ADTStreamLog complexity = %s, want %s",
			c, metaengine.ComplexityOLogN)
	}

	if profile.IsDegraded(metaengine.ADTStreamLog) {
		t.Error("ADTStreamLog should NOT be degraded on Dgraph")
	}
}
