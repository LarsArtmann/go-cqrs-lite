package enginetest

import (
	"context"
	"reflect"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RunSeqSeekableStreamLogTest exercises the SeqSeekableStreamLog contract
// against the default "events" collection. The engine must already implement
// the capability; the caller is responsible for closing the engine.
func RunSeqSeekableStreamLogTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	RunSeqSeekableStreamLogTestIn(t, eng, "events_seq")
}

// RunSeqSeekableStreamLogTestIn is RunSeqSeekableStreamLogTest with a
// caller-chosen collection, for engines whose storage outlives a single test.
//
// The contract:
//  1. JournalReadAllWithSeq matches JournalReadAll (same values, same order)
//     with strictly increasing Seq tokens starting at >= 1
//  2. JournalReadFromSeq(0, 0) returns the full journal
//  3. JournalReadFromSeq(token of entry k) returns exactly the suffix after k
//     — including when other collections interleave appends between this
//     collection's entries (shared-autoincrement engines produce gapped,
//     non-contiguous tokens)
//  4. JournalReadFromSeq honors limit
//  5. A cursor past the last entry returns an empty result
//  6. On the dense journal, token resumption agrees with position-based
//     JournalReadFrom
func RunSeqSeekableStreamLogTestIn(t *testing.T, eng metaengine.Engine, col string) {
	t.Helper()

	// Per-run suffix: the contract starts from an empty journal, which only
	// holds when repeated invocations get fresh collections.
	col = ScopedCollection(col)

	ss, ok := eng.(metaengine.SeqSeekableStreamLog)
	if !ok {
		t.Fatalf("engine %T does not implement SeqSeekableStreamLog", eng)
	}

	slb, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatalf("engine %T does not implement StreamLogBackend", eng)
	}

	ctx := context.Background()

	// Empty journal: both reads return empty, not nil-not-empty errors.
	emptyAll, err := ss.JournalReadAllWithSeq(ctx, col)
	if err != nil || len(emptyAll) != 0 {
		t.Fatalf("JournalReadAllWithSeq on empty collection = %v, %v; want empty", emptyAll, err)
	}

	emptyFrom, err := ss.JournalReadFromSeq(ctx, col, 0, 0)
	if err != nil || len(emptyFrom) != 0 {
		t.Fatalf("JournalReadFromSeq on empty collection = %v, %v; want empty", emptyFrom, err)
	}

	// Interleave appends: col, other, col, other, col. On engines with a
	// shared seq counter this makes col's tokens gapped (e.g. 1, 3, 5) —
	// exactly the case where naive position arithmetic mis-cursors.
	other := col + "_interleave"

	steps := []struct {
		col     string
		stream  string
		entries []any
	}{
		{col, "s1", []any{"e1"}},
		{other, "x1", []any{"noise-1"}},
		{col, "s1", []any{"e2"}},
		{other, "x1", []any{"noise-2"}},
		{col, "s2", []any{"e3", "e4"}},
	}

	for _, step := range steps {
		if err := slb.StreamAppend(ctx, step.col, step.stream, step.entries); err != nil {
			t.Fatalf("StreamAppend %s/%s: %v", step.col, step.stream, err)
		}
	}

	// 1. JournalReadAllWithSeq matches JournalReadAll with increasing tokens.
	all, err := slb.JournalReadAll(ctx, col)
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}

	withSeq, err := ss.JournalReadAllWithSeq(ctx, col)
	if err != nil {
		t.Fatalf("JournalReadAllWithSeq: %v", err)
	}

	if len(withSeq) != len(all) {
		t.Fatalf(
			"JournalReadAllWithSeq returned %d entries, JournalReadAll %d",
			len(withSeq),
			len(all),
		)
	}

	for i, entry := range withSeq {
		if !reflect.DeepEqual(entry.Value, all[i]) {
			t.Fatalf("entry %d value = %v, want %v (JournalReadAll order)", i, entry.Value, all[i])
		}

		if entry.Seq < 1 {
			t.Fatalf("entry %d Seq = %d, want >= 1", i, entry.Seq)
		}

		if i > 0 && entry.Seq <= withSeq[i-1].Seq {
			t.Fatalf(
				"Seq not strictly increasing at %d: %d after %d",
				i,
				entry.Seq,
				withSeq[i-1].Seq,
			)
		}
	}

	// 2. Cursor 0 returns the full journal.
	fromStart, err := ss.JournalReadFromSeq(ctx, col, 0, 0)
	if err != nil {
		t.Fatalf("JournalReadFromSeq(0, 0): %v", err)
	}

	if !reflect.DeepEqual(fromStart, withSeq) {
		t.Fatalf("JournalReadFromSeq(0, 0) = %v, want full journal %v", fromStart, withSeq)
	}

	// 3. Token of entry k resumes exactly after k, for every k.
	for k := range withSeq {
		suffix, err := ss.JournalReadFromSeq(ctx, col, withSeq[k].Seq, 0)
		if err != nil {
			t.Fatalf("JournalReadFromSeq(token %d): %v", withSeq[k].Seq, err)
		}

		want := withSeq[k+1:]

		if len(suffix) != len(want) {
			t.Fatalf("token %d: got %d entries, want %d", withSeq[k].Seq, len(suffix), len(want))
		}

		for i := range suffix {
			if suffix[i].Seq != want[i].Seq || !reflect.DeepEqual(suffix[i].Value, want[i].Value) {
				t.Fatalf("token %d: entry %d = {%d %v}, want {%d %v}",
					withSeq[k].Seq, i, suffix[i].Seq, suffix[i].Value, want[i].Seq, want[i].Value)
			}
		}
	}

	// 4. limit is honored on token resumption.
	limited, err := ss.JournalReadFromSeq(ctx, col, withSeq[0].Seq, 2)
	if err != nil {
		t.Fatalf("JournalReadFromSeq limited: %v", err)
	}

	if len(limited) != 2 || limited[0].Seq != withSeq[1].Seq || limited[1].Seq != withSeq[2].Seq {
		t.Fatalf("JournalReadFromSeq(token %d, 2) = %v, want entries 1-2", withSeq[0].Seq, limited)
	}

	// 5. Cursor past the end returns empty.
	past, err := ss.JournalReadFromSeq(ctx, col, withSeq[len(withSeq)-1].Seq, 0)
	if err != nil || len(past) != 0 {
		t.Fatalf("JournalReadFromSeq(last token) = %v, %v; want empty", past, err)
	}

	// 6. Dense-journal agreement with position-based JournalReadFrom: the
	// journal has no deletions, so position k and the token of entry k-1
	// must resume at the same place.
	for k := 1; k <= len(withSeq); k++ {
		byPos, err := slb.JournalReadFrom(ctx, col, int64(k), 0)
		if err != nil {
			t.Fatalf("JournalReadFrom(%d): %v", k, err)
		}

		cursor := int64(0)
		if k > 0 {
			cursor = withSeq[k-1].Seq
		}

		byToken, err := ss.JournalReadFromSeq(ctx, col, cursor, 0)
		if err != nil {
			t.Fatalf("JournalReadFromSeq(token %d): %v", cursor, err)
		}

		if len(byPos) != len(byToken) {
			t.Fatalf(
				"position %d: %d entries vs token %d: %d entries",
				k,
				len(byPos),
				cursor,
				len(byToken),
			)
		}

		for i := range byPos {
			if !reflect.DeepEqual(byPos[i], byToken[i].Value) {
				t.Fatalf(
					"position %d entry %d = %v, token resume = %v",
					k,
					i,
					byPos[i],
					byToken[i].Value,
				)
			}
		}
	}
}
