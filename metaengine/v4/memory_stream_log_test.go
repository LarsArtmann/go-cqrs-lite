package metaengine

import (
	"context"
	"testing"
)

func TestStreamLogBackend_MemoryRoundtrip(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer eng.Close()

	be := eng.(StreamLogBackend)

	ctx := context.Background()
	col := "events"
	streamA := "task-1"
	streamB := "task-2"

	// Append to stream A
	if err := be.StreamAppend(ctx, col, streamA, []any{"evt-1", "evt-2"}); err != nil {
		t.Fatalf("StreamAppend A: %v", err)
	}

	// Append to stream B
	if err := be.StreamAppend(ctx, col, streamB, []any{"evt-3"}); err != nil {
		t.Fatalf("StreamAppend B: %v", err)
	}

	// StreamRead returns only the stream's values
	got, err := be.StreamRead(ctx, col, streamA)
	if err != nil {
		t.Fatalf("StreamRead A: %v", err)
	}
	if len(got) != 2 || got[0] != "evt-1" || got[1] != "evt-2" {
		t.Fatalf("StreamRead A = %v, want [evt-1 evt-2]", got)
	}

	// StreamVersion
	ver, err := be.StreamVersion(ctx, col, streamA)
	if err != nil {
		t.Fatalf("StreamVersion A: %v", err)
	}
	if ver != 2 {
		t.Fatalf("StreamVersion A = %d, want 2", ver)
	}

	verB, _ := be.StreamVersion(ctx, col, streamB)
	if verB != 1 {
		t.Fatalf("StreamVersion B = %d, want 1", verB)
	}

	// JournalReadAll returns all values across streams in append order
	all, err := be.JournalReadAll(ctx, col)
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("JournalReadAll len = %d, want 3", len(all))
	}
	if all[0] != "evt-1" || all[1] != "evt-2" || all[2] != "evt-3" {
		t.Fatalf("JournalReadAll = %v, want [evt-1 evt-2 evt-3]", all)
	}

	// JournalReadFrom: after seq 1 (skip evt-1), limit 10
	from, err := be.JournalReadFrom(ctx, col, 1, 10)
	if err != nil {
		t.Fatalf("JournalReadFrom: %v", err)
	}
	if len(from) != 2 || from[0] != "evt-2" || from[1] != "evt-3" {
		t.Fatalf("JournalReadFrom(1) = %v, want [evt-2 evt-3]", from)
	}

	// JournalReadFrom with limit
	limited, _ := be.JournalReadFrom(ctx, col, 0, 2)
	if len(limited) != 2 || limited[0] != "evt-1" || limited[1] != "evt-2" {
		t.Fatalf("JournalReadFrom(0, 2) = %v, want [evt-1 evt-2]", limited)
	}
}

func TestStreamLogBackend_EmptyStream(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer eng.Close()

	ctx := context.Background()

	be := eng.(StreamLogBackend)

	got, err := be.StreamRead(ctx, "events", "nonexistent")
	if err != nil {
		t.Fatalf("StreamRead nonexistent: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("StreamRead nonexistent = %v, want empty", got)
	}

	ver, _ := be.StreamVersion(ctx, "events", "nonexistent")
	if ver != 0 {
		t.Fatalf("StreamVersion nonexistent = %d, want 0", ver)
	}

	all, _ := be.JournalReadAll(ctx, "events")
	if len(all) != 0 {
		t.Fatalf("JournalReadAll empty = %v, want empty", all)
	}
}
