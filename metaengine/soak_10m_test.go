package metaengine_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestSoak_MemoryBounded_10M verifies that processing 10M events into a bounded
// set of keys does not cause unbounded memory growth. The memory engine's Map
// ADT is O(unique keys), not O(total events).
//
// The test samples heap at regular intervals (every 2M events) to verify the
// growth curve is flat — not linear with events. It also verifies correctness
// by checking accumulated totals for all keys after processing.
//
// Skips in -short mode. Skips when SOAK_SKIP_10M=1 (for CI that cannot afford
// the runtime). Runtime: ~5s without -race, ~25s with -race.
func TestSoak_MemoryBounded_10M(t *testing.T) {
	if testing.Short() {
		t.Skip("10M soak test: skips in -short mode")
	}

	if os.Getenv("SOAK_SKIP_10M") == "1" {
		t.Skip("10M soak test: skipped by SOAK_SKIP_10M=1")
	}

	t.Parallel()

	type updateEvent struct {
		Key   string
		Value int64
	}
	type lookup struct {
		Key string
	}
	type state struct {
		Key   string
		Total int64
	}

	q := metaengine.Query[lookup, state](
		"counters-10m",
		metaengine.On(updateEvent{}, func(e updateEvent) (string, state) {
			return e.Key, state{Key: e.Key, Total: e.Value}
		}),
		metaengine.On(updateEvent{}, func(e updateEvent, prev state) state {
			prev.Total += e.Value

			return prev
		}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	ctx := context.Background()

	const (
		numEvents   = 10_000_000
		numKeys     = 1_000      // 10K updates per key — memory bounded by numKeys
		sampleEvery = 2_000_000 // sample heap every 2M events
	)

	// Pre-generate keys to avoid fmt.Sprintf allocations in the hot loop.
	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("k-%d", i)
	}

	// Pre-compute expected totals: key k receives values i where i%numKeys == k.
	expected := make([]int64, numKeys)
	for i := range numEvents {
		expected[i%numKeys] += int64(i)
	}

	// Baseline heap measurement after all setup allocations.
	runtime.GC()

	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	type heapSample struct {
		events    int
		heapBytes uint64
	}

	samples := make([]heapSample, 0, numEvents/sampleEvery+1)
	samples = append(samples, heapSample{events: 0, heapBytes: baseline.HeapAlloc})

	// Process events, sampling heap at regular intervals.
	for i := range numEvents {
		if err := store.Apply(
			ctx,
			"updateEvent",
			updateEvent{Key: keys[i%numKeys], Value: int64(i)},
		); err != nil {
			t.Fatalf("Apply %d: %v", i, err)
		}

		// Sample at regular intervals (skip i==0).
		if i > 0 && i%sampleEvery == 0 {
			runtime.GC()

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			samples = append(samples, heapSample{events: i + 1, heapBytes: m.HeapAlloc})
		}
	}

	// Final measurement.
	runtime.GC()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	totalGrowth := int64(after.HeapAlloc) - int64(baseline.HeapAlloc)

	// Threshold: memory should be O(numKeys), not O(numEvents).
	// 1000 keys → ~100KB of map entries. Allow generous headroom for
	// GC pressure, map overhead, and parallel test load.
	maxGrowth := int64(15 * 1024 * 1024) // 15MB
	if raceEnabled {
		maxGrowth *= 5 // 75MB under -race
	}

	if totalGrowth > maxGrowth {
		t.Errorf("heap grew %d bytes after %d events with %d keys (max %d)",
			totalGrowth, numEvents, numKeys, maxGrowth)
	}

	// Verify the growth curve is flat: sustained growth in the last segment
	// indicates a leak, not just initial ramp-up.
	if len(samples) >= 3 {
		lastSeg := int64(samples[len(samples)-1].heapBytes) -
			int64(samples[len(samples)-2].heapBytes)
		maxLastSeg := int64(2 * 1024 * 1024) // 2MB
		if raceEnabled {
			maxLastSeg *= 5 // 10MB under -race
		}

		if lastSeg > maxLastSeg {
			t.Errorf("sustained heap growth in last segment: %d bytes (max %d) — possible leak",
				lastSeg, maxLastSeg)
		}
	}

	// Correctness: verify all keys have the correct accumulated total.
	var mismatches int

	for k := range numKeys {
		result, err := metaengine.ExecuteTyped[lookup, state](
			ctx, store, lookup{Key: keys[k]},
		)
		if err != nil {
			t.Fatalf("ExecuteTyped key %d: %v", k, err)
		}

		if result.Total != expected[k] {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("key %s: expected total %d, got %d (delta=%d)",
					keys[k], expected[k], result.Total, result.Total-expected[k])
			}
		}
	}

	if mismatches > 5 {
		t.Errorf("...and %d more mismatches", mismatches-5)
	}

	// Log growth curve for diagnostics.
	t.Logf("10M soak: %d events, %d keys, %d bytes heap growth (%.1f MB)",
		numEvents, numKeys, totalGrowth, float64(totalGrowth)/1024/1024)

	for _, s := range samples {
		t.Logf("  %8d events → %.1f MB heap", s.events, float64(s.heapBytes)/1024/1024)
	}
}
