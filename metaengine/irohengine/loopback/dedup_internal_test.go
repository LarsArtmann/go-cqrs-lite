package loopback

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// Pins the op-dedup window semantics of markSeen. The window is a bounded
// dedup.Ring with graceful eviction of the oldest ID, not a map that resets
// wholesale: the newest IDs always survive, so a redelivery inside the live
// window is deduplicated without the full reset gap the hand-rolled map had.
// These tests make the bounded-eviction contract explicit. The quic twin keeps
// the identical contract in dedup_parity_test.go, and both rings are built
// from the shared irohengine.DefaultDedupCapacity so the windows cannot drift.
func TestMarkSeen_DedupWindow(t *testing.T) {
	tr := &LoopbackTransport{dedupRing: dedup.NewRing(irohengine.DefaultDedupCapacity)}

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

// TestMarkSeen_EvictsOldestNotAll pins the property the reset-based map lacked:
// at capacity, exactly the oldest ID is forgotten — recent IDs stay deduplicated.
func TestMarkSeen_EvictsOldestNotAll(t *testing.T) {
	const capacity = 4

	tr := &LoopbackTransport{dedupRing: dedup.NewRing(capacity)}

	const oldest = "op-oldest"

	tr.markSeen(oldest)
	for i := range capacity - 1 {
		tr.markSeen(fmt.Sprintf("op-%02d", i))
	}

	// Capacity reached: the next distinct ID evicts exactly the oldest.
	tr.markSeen("op-next")

	if got := tr.dedupRing.Len(); got > capacity {
		t.Fatalf("dedup ring holds %d entries, want <= %d", got, capacity)
	}

	if tr.markSeen("op-next") {
		t.Fatal("recently added op-next = true, want false (still deduplicated)")
	}

	if !tr.markSeen(oldest) {
		t.Fatal("markSeen(oldest) = false; evicted entry should be forgotten")
	}
}

// TestMarkSeen_BoundedOverflow proves memory stays bounded across far more IDs
// than the capacity, while every recent ID remains deduplicated.
func TestMarkSeen_BoundedOverflow(t *testing.T) {
	tr := &LoopbackTransport{dedupRing: dedup.NewRing(irohengine.DefaultDedupCapacity)}

	for i := range irohengine.DefaultDedupCapacity * 2 {
		tr.markSeen(fmt.Sprintf("fill-%06d", i))
	}

	if got := tr.dedupRing.Len(); got > irohengine.DefaultDedupCapacity {
		t.Fatalf("dedup ring grew to %d entries, want <= %d", got, irohengine.DefaultDedupCapacity)
	}

	if tr.markSeen(fmt.Sprintf("fill-%06d", irohengine.DefaultDedupCapacity*2-1)) {
		t.Fatal("most recent ID = true, want false (still deduplicated)")
	}
}
