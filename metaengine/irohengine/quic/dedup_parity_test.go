package quic

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// Dedup parity with the loopback transport twin: both transports build their
// op-dedup ring from the ONE shared irohengine.DefaultDedupCapacity (no
// per-transport const that could drift) and both markSeen implementations
// honor the identical window contract — bounded ring, graceful eviction of the
// oldest ID, no wholesale reset. loopback pins the same contract in its
// dedup_internal_test.go; this file is the quic side of that parity.
func TestDedupParity_SharedCapacityConst(t *testing.T) {
	if DefaultDedupCapacity != irohengine.DefaultDedupCapacity {
		t.Fatalf("quic.DefaultDedupCapacity = %d, want irohengine.DefaultDedupCapacity (%d)",
			DefaultDedupCapacity, irohengine.DefaultDedupCapacity)
	}
}

// TestRing_ProductionCapacity10K pins ring behavior at exactly the shared
// production capacity: duplicate IDs deduplicate, and after overflowing the
// window memory stays bounded while the most recent IDs remain deduplicated.
func TestRing_ProductionCapacity10K(t *testing.T) {
	tr := &QuicTransport{dedupRing: dedup.NewRing(DefaultDedupCapacity)}

	if !tr.markSeen("op-1") {
		t.Fatal("first markSeen(op-1) = false, want true")
	}

	if tr.markSeen("op-1") {
		t.Fatal("second markSeen(op-1) = true, want false (dedup)")
	}

	for i := range DefaultDedupCapacity * 2 {
		tr.markSeen(fmt.Sprintf("fill-%06d", i))
	}

	if got := tr.dedupRing.Len(); got > DefaultDedupCapacity {
		t.Fatalf("dedup ring grew to %d entries, want <= %d", got, DefaultDedupCapacity)
	}

	if tr.markSeen(fmt.Sprintf("fill-%06d", DefaultDedupCapacity*2-1)) {
		t.Fatal("most recent ID = true, want false (still deduplicated)")
	}
}
