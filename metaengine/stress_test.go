package metaengine_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// V4: Stress test — seed 100K events and verify scan correctness + point-lookup
// accuracy + memory stability. This proves metaengine handles production-scale
// volumes without OOM or correctness regressions.
func TestStress_100KEvents(t *testing.T) { //nolint:tparallel // subtests share the parent's store
	t.Parallel()

	const N = 100_000

	store, reader := setupBenchStore(t, N, true)
	defer store.Close()

	ctx := context.Background()

	// ─── Point-lookup correctness ───
	t.Run("PointLookup", func(t *testing.T) {
		// Sequential: shares the parent's store.
		// Verify 100 random IDs return correct data.
		for i := 0; i < 100; i++ {
			idx := i * (N / 100) // spread across the full range
			key := fmt.Sprintf("item-%06d", idx)

			item, ok, err := reader.Get(ctx, key)
			if err != nil {
				t.Fatalf("Get(%s): %v", key, err)
			}

			if !ok {
				t.Fatalf("Get(%s): not found", key)
			}

			if item.ID != key {
				t.Errorf("Get(%s) returned ID=%s", key, item.ID)
			}
		}
	})

	// ─── Filtered scan correctness ───
	t.Run("FilteredScan", func(t *testing.T) {
		// Sequential: shares the parent's store.
		openItems, err := reader.Scan(ctx,
			metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
			metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Scan open: %v", err)
		}

		// 2/3 should be open (i%3 != 0).
		expectedClosed := N/3 + 1 // includes index 0
		expectedOpen := N - expectedClosed
		if len(openItems) != expectedOpen {
			t.Errorf("open items = %d, want %d", len(openItems), expectedOpen)
		}

		closedItems, err := reader.Scan(ctx,
			metaengine.WithFilter("Status", metaengine.FilterEq, "closed"),
			metaengine.WithLimit(0))
		if err != nil {
			t.Fatalf("Scan closed: %v", err)
		}

		// 1/3 should be closed (i%3 == 0).
		if len(closedItems) != expectedClosed {
			t.Errorf("closed items = %d, want %d", len(closedItems), expectedClosed)
		}

		totalScanned := len(openItems) + len(closedItems)
		if totalScanned != N {
			t.Errorf("total scanned = %d, want %d", totalScanned, N)
		}
	})

	// ─── Sorted scan returns priority-ordered results ───
	t.Run("SortedScan", func(t *testing.T) {
		// Sequential: shares the parent's store.
		results, err := reader.Scan(ctx,
			metaengine.WithFilter("Status", metaengine.FilterEq, "closed"),
			metaengine.WithSort("Priority", true))
		if err != nil {
			t.Fatalf("Scan sorted: %v", err)
		}

		// Verify descending order.
		for i := 1; i < len(results); i++ {
			if results[i].Priority > results[i-1].Priority {
				t.Errorf("sorted scan not DESC at [%d]: %d > %d",
					i, results[i].Priority, results[i-1].Priority)
			}
		}
	})

	// ─── Memory stability ───
	t.Run("MemoryStability", func(t *testing.T) {
		// Sequential: shares the parent's store.
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)

		// Run 10 full scans (stress the allocator).
		for range 10 {
			_, err := reader.Scan(ctx, metaengine.WithLimit(0))
			if err != nil {
				t.Fatalf("full Scan: %v", err)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&after)

		// Heap should not grow unboundedly after repeated scans.
		growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
		if growth > 50*1024*1024 { // 50MB tolerance
			t.Errorf("heap grew by %d MB after 10 scans (before=%d, after=%d)",
				growth/1024/1024, before.HeapAlloc/1024/1024, after.HeapAlloc/1024/1024)
		}
	})
}
