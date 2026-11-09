package metaengine

import (
	"context"
	"testing"
)

// TestMemory_JournalReadFromSeq_GapTolerance verifies the token contract's
// gap tolerance: after removing a journal entry (simulating a future
// deletion), resuming from a token before the gap still delivers exactly the
// remaining suffix — no skipped and no re-delivered entries. Position
// arithmetic (afterSeq + i) would mis-cursor here; seq > cursor cannot.
func TestMemory_JournalReadFromSeq_GapTolerance(t *testing.T) {
	t.Parallel()

	m := NewMemoryEngine().(*memoryEngine)
	defer m.Close()

	ctx := context.Background()

	if err := m.StreamAppend(ctx, "gaps", "s1", []any{"a", "b", "c", "d"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	// Simulate deletion of entry "b" (seq 2): the journal becomes
	// [seq:1 a, seq:3 c, seq:4 d] — a gap where position != seq.
	j := m.data.streamJournal["gaps"]
	m.data.streamJournal["gaps"] = append(j[:1], j[2:]...)

	fromSeq1, err := m.JournalReadFromSeq(ctx, "gaps", 1, 0)
	if err != nil {
		t.Fatalf("JournalReadFromSeq(1): %v", err)
	}

	if len(fromSeq1) != 2 || fromSeq1[0].Value != "c" || fromSeq1[1].Value != "d" {
		t.Fatalf(
			"JournalReadFromSeq(1) = %v, want [c d] (gap skipped, nothing re-delivered)",
			fromSeq1,
		)
	}

	if fromSeq1[0].Seq != 3 || fromSeq1[1].Seq != 4 {
		t.Fatalf("gap tokens = [%d %d], want [3 4]", fromSeq1[0].Seq, fromSeq1[1].Seq)
	}

	// Resuming from the token of the last entry before a gap mid-journal.
	fromSeq3, err := m.JournalReadFromSeq(ctx, "gaps", 3, 0)
	if err != nil {
		t.Fatalf("JournalReadFromSeq(3): %v", err)
	}

	if len(fromSeq3) != 1 || fromSeq3[0].Value != "d" {
		t.Fatalf("JournalReadFromSeq(3) = %v, want [d]", fromSeq3)
	}
}
