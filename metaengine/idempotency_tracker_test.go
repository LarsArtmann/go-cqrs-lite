package metaengine

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestIdempotencyTracker_BoundedEvictsOldest(t *testing.T) {
	t.Parallel()

	tracker := newIdempotencyTracker(8)

	for i := range 16 {
		if tracker.CheckAndRecord(fmt.Sprintf("evt-%02d", i)) {
			t.Fatalf("evt-%02d flagged as duplicate on first sight", i)
		}
	}

	if got := tracker.Len(); got != 8 {
		t.Fatalf("Len() = %d, want 8 (capacity bound)", got)
	}

	// The newest 8 IDs are still deduplicated. Checked first: these calls do
	// not mutate the ring, unlike the eviction probe below.
	for i := 8; i < 16; i++ {
		if !tracker.CheckAndRecord(fmt.Sprintf("evt-%02d", i)) {
			t.Fatalf("evt-%02d not deduplicated while inside the window", i)
		}
	}

	// The oldest 8 IDs were evicted: they are no longer deduplicated (and
	// re-recording evt-00 evicts evt-08, which is expected ring behavior).
	if tracker.CheckAndRecord("evt-00") {
		t.Fatal("evt-00 still deduplicated after eviction — ring is not bounded")
	}
}

// TestIdempotencyTracker_MemoryBoundedUnderManyIDs feeds 1M event IDs
// through a small tracker: tracked IDs must stay at capacity regardless of
// how many IDs flow through (the unbounded sync.Map this replaced leaked all
// of them).
func TestIdempotencyTracker_MemoryBoundedUnderManyIDs(t *testing.T) {
	t.Parallel()

	const capacity = 1024

	tracker := newIdempotencyTracker(capacity)

	for i := range 1_000_000 {
		tracker.CheckAndRecord(fmt.Sprintf("evt-%d", i))
	}

	if got := tracker.Len(); got != capacity {
		t.Fatalf("Len() = %d after 1M IDs, want %d — tracker is not memory-bounded", got, capacity)
	}
}

func TestIdempotencyTracker_UnboundedLegacy(t *testing.T) {
	t.Parallel()

	tracker := newIdempotencyTracker(0)

	for i := range 100 {
		if tracker.CheckAndRecord(fmt.Sprintf("evt-%d", i)) {
			t.Fatalf("evt-%d flagged as duplicate on first sight", i)
		}
	}

	if got := tracker.Len(); got != 100 {
		t.Fatalf("Len() = %d, want 100 (legacy unbounded keeps everything)", got)
	}

	for i := range 100 {
		if !tracker.CheckAndRecord(fmt.Sprintf("evt-%d", i)) {
			t.Fatalf("evt-%d not deduplicated in unbounded mode", i)
		}
	}
}

func TestIdempotencyTracker_ConcurrentExactlyOnce(t *testing.T) {
	t.Parallel()

	const (
		ids     = 32
		workers = 8
		rounds  = 10
	)

	// Capacity comfortably exceeds the ID set, so eviction cannot interfere.
	tracker := newIdempotencyTracker(64)

	var duplicates [ids]atomic.Int32

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			local := make([]int32, ids)

			for range rounds {
				for i := range ids {
					if tracker.CheckAndRecord(fmt.Sprintf("evt-%d", i)) {
						local[i]++
					}
				}
			}

			for i := range ids {
				duplicates[i].Add(local[i])
			}
		}()
	}

	wg.Wait()

	for i := range ids {
		if got := duplicates[i].Load(); got != workers*rounds-1 {
			t.Fatalf(
				"evt-%d reported duplicate %d times, want exactly %d (first call records, rest dedup)",
				i,
				got,
				workers*rounds-1,
			)
		}
	}
}
