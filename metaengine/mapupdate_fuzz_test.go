package metaengine

import (
	"context"
	"sync"
	"testing"
)

// FuzzMapUpdate_ConcurrentCounter verifies that concurrent MapUpdate calls
// on the same key produce a consistent final value (no lost updates).
// The fuzz input controls the number of goroutines.
func FuzzMapUpdate_ConcurrentCounter(f *testing.F) {
	f.Add(2)  // 2 goroutines
	f.Add(10) // 10 goroutines
	f.Add(50) // 50 goroutines

	f.Fuzz(func(t *testing.T, goroutines int) {
		if goroutines < 1 || goroutines > 200 {
			t.Skip("goroutine count out of testable range")
		}

		ctx := context.Background()
		eng := NewMemoryEngine()
		defer eng.Close()

		mb := eng.(MapBackend)
		mu := eng.(MapUpdater)

		const col = "counters"
		const key = "total"

		if err := mb.MapSet(ctx, col, key, int64(0)); err != nil {
			t.Fatalf("MapSet: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()

				err := mu.MapUpdate(ctx, col, key, func(prev any) any {
					return prev.(int64) + 1
				})
				if err != nil {
					t.Errorf("MapUpdate: %v", err)
				}
			}()
		}

		wg.Wait()

		val, _, err := mb.MapGet(ctx, col, key)
		if err != nil {
			t.Fatalf("MapGet: %v", err)
		}

		got := val.(int64)
		if got != int64(goroutines) {
			t.Errorf("after %d concurrent increments: expected %d, got %d (lost updates)",
				goroutines, goroutines, got)
		}
	})
}

// FuzzMapUpdate_CreateOrUpdate verifies that MapUpdate correctly handles
// the "create if absent, update if present" pattern for arbitrary initial values.
func FuzzMapUpdate_CreateOrUpdate(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(42))
	f.Add(int64(-1))

	f.Fuzz(func(t *testing.T, initial int64) {
		ctx := context.Background()
		eng := NewMemoryEngine()
		defer eng.Close()

		mb := eng.(MapBackend)
		mu := eng.(MapUpdater)

		const col = "items"
		const key = "k1"

		if err := mb.MapSet(ctx, col, key, initial); err != nil {
			t.Fatalf("MapSet: %v", err)
		}

		err := mu.MapUpdate(ctx, col, key, func(prev any) any {
			if prev == nil {
				return int64(1)
			}

			v, ok := prev.(int64)
			if !ok {
				return int64(1)
			}

			return v * 2
		})
		if err != nil {
			t.Fatalf("MapUpdate: %v", err)
		}

		val, _, err := mb.MapGet(ctx, col, key)
		if err != nil {
			t.Fatalf("MapGet: %v", err)
		}

		got := val.(int64)
		expected := initial * 2

		if got != expected {
			t.Errorf("after doubling %d: expected %d, got %d", initial, expected, got)
		}
	})
}
