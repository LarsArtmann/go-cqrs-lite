package loopback

import (
	"fmt"
	"testing"
)

// Pins the op-dedup window semantics of markSeen. The dedup set trades
// correctness precision for bounded memory: beyond 10,000 entries the whole
// set RESETS (not a ring/LFU eviction), so an op ID seen before the reset is
// forgotten and would be re-applied on a later redelivery. These tests make
// both halves of that contract explicit.
func TestMarkSeen_DedupWindow(t *testing.T) {
	tr := &LoopbackTransport{}
	tr.dedupSeen = make(map[string]struct{})

	if !tr.markSeen("op-1") {
		t.Fatal("first markSeen(op-1) = false, want true")
	}

	if tr.markSeen("op-1") {
		t.Fatal("second markSeen(op-1) = true, want false (dedup)")
	}

	if !tr.markSeen("op-2") {
		t.Fatal("markSeen(op-2) = false, want true (distinct ID)")
	}
}

func TestMarkSeen_ResetBeyondWindow(t *testing.T) {
	tr := &LoopbackTransport{}
	tr.dedupSeen = make(map[string]struct{})

	const window = 10000

	// Fill the set past the reset threshold, remembering the FIRST id —
	// it is the one the reset will forget.
	const oldID = "op-0000"
	tr.markSeen(oldID)
	for i := range window {
		tr.markSeen(fmt.Sprintf("fill-%06d", i))
	}

	if got := len(tr.dedupSeen); got > window+1 {
		t.Fatalf("dedup set grew to %d entries, want bounded ~%d", got, window)
	}

	// Old IDs are forgotten after the reset: redelivery of pre-reset ops
	// re-applies. This is the documented tradeoff, not a bug.
	if !tr.markSeen(oldID) {
		t.Fatal("markSeen(oldID) after reset = false; reset should have forgotten it")
	}
}
