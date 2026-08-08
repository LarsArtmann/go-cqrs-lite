package metaengine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestMemoryEngine_ConcurrentMapAccess proves that the MemoryEngine's
// internal RWMutex correctly serializes concurrent MapBackend operations
// (MapSet, MapGet, MapDelete). Run with -race to detect data races.
//
// This test exercises the engine directly — not through the Store/Plan
// layer — so it catches synchronization bugs at the lowest level.
func TestMemoryEngine_ConcurrentMapAccess(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer DeferClose(eng)

	mb, ok := eng.(MapBackend)
	if !ok {
		t.Fatal("memoryEngine does not implement MapBackend")
	}

	ctx := context.Background()
	const goroutines = 20
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	var ops atomic.Int64

	for i := range goroutines {
		go func(workerID int) {
			defer wg.Done()

			for j := range iterations {
				key := fmt.Sprintf("worker-%d-key-%d", workerID, j%10)
				val := fmt.Sprintf("val-%d-%d", workerID, j)

				switch j % 3 {
				case 0:
					_ = mb.MapSet(ctx, "concurrent-test", key, val)
				case 1:
					_, _, _ = mb.MapGet(ctx, "concurrent-test", key)
				case 2:
					_ = mb.MapDelete(ctx, "concurrent-test", key)
				}

				ops.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if got := ops.Load(); got != goroutines*iterations {
		t.Fatalf("expected %d total ops, got %d", goroutines*iterations, got)
	}
}

// TestMemoryEngine_ConcurrentCounterAccess proves that concurrent
// CounterIncrement and CounterGet calls are race-free under -race.
func TestMemoryEngine_ConcurrentCounterAccess(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer DeferClose(eng)

	cb, ok := eng.(CounterBackend)
	if !ok {
		t.Fatal("memoryEngine does not implement CounterBackend")
	}

	ctx := context.Background()
	const goroutines = 20
	const increments = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range increments {
				_ = cb.CounterIncrement(ctx, "race-counter", Delta{"count": 1})
			}
		}()
	}

	wg.Wait()

	counts, err := cb.CounterGet(ctx, "race-counter")
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	expected := int64(goroutines * increments)
	if counts["count"] != expected {
		t.Fatalf("expected counter %d, got %d", expected, counts["count"])
	}
}

// TestMemoryEngine_ConcurrentMixedBackends proves that concurrent
// operations across different backend types (Map + Set + Counter) on
// the same MemoryEngine are race-free.
func TestMemoryEngine_ConcurrentMixedBackends(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer DeferClose(eng)

	ctx := context.Background()
	mb, _ := eng.(MapBackend)
	sb, _ := eng.(SetBackend)
	cb, _ := eng.(CounterBackend)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			_ = mb.MapSet(ctx, "map-col", fmt.Sprintf("k%d", id), id)
		}(i)

		go func(id int) {
			defer wg.Done()
			_ = sb.SetAdd(ctx, "set-col", fmt.Sprintf("member-%d", id))
		}(i)

		go func(id int) {
			defer wg.Done()
			_ = cb.CounterIncrement(ctx, "counter-col", Delta{"c": int64(id)})
		}(i)
	}

	wg.Wait()
}
