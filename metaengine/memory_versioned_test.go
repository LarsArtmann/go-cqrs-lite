package metaengine

import (
	"context"
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestMemoryEngine_VersionedStorage(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngineWithVersioning()
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(MapBackend)

	// Write v1
	if err := mb.MapSet(
		ctx,
		"users",
		"u1",
		map[string]any{"name": "Alice", "v": float64(1)},
	); err != nil {
		t.Fatal(err)
	}

	t1 := time.Now()
	time.Sleep(1 * time.Millisecond)

	// Write v2
	if err := mb.MapSet(
		ctx,
		"users",
		"u1",
		map[string]any{"name": "Alice", "v": float64(2)},
	); err != nil {
		t.Fatal(err)
	}

	t2 := time.Now()
	time.Sleep(1 * time.Millisecond)

	// Delete
	if err := mb.MapDelete(ctx, "users", "u1"); err != nil {
		t.Fatal(err)
	}

	t3 := time.Now()

	vs := eng.(VersionedStorage)

	// AsOf t1 → should return v1
	val, err := vs.MapGetAsOf(ctx, "users", "u1", t1)
	if err != nil {
		t.Fatalf("MapGetAsOf(t1): %v", err)
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("MapGetAsOf(t1): expected map, got %T", val)
	}

	if m["v"] != float64(1) {
		t.Errorf("MapGetAsOf(t1): v = %v, want 1", m["v"])
	}

	// AsOf t2 → should return v2
	val, err = vs.MapGetAsOf(ctx, "users", "u1", t2)
	if err != nil {
		t.Fatalf("MapGetAsOf(t2): %v", err)
	}

	m, ok = val.(map[string]any)
	if !ok {
		t.Fatalf("MapGetAsOf(t2): expected map, got %T", val)
	}

	if m["v"] != float64(2) {
		t.Errorf("MapGetAsOf(t2): v = %v, want 2", m["v"])
	}

	// AsOf t3 → should return ErrNotFound (deleted)
	_, err = vs.MapGetAsOf(ctx, "users", "u1", t3)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("MapGetAsOf(t3): err = %v, want ErrNotFound", err)
	}

	// MapExistsAsOf checks
	exists, _ := vs.MapExistsAsOf(ctx, "users", "u1", t1)
	if !exists {
		t.Error("MapExistsAsOf(t1): expected true")
	}

	exists, _ = vs.MapExistsAsOf(ctx, "users", "u1", t3)
	if exists {
		t.Error("MapExistsAsOf(t3): expected false (deleted)")
	}

	// Non-existent key
	exists, _ = vs.MapExistsAsOf(ctx, "users", "nonexistent", t1)
	if exists {
		t.Error("MapExistsAsOf(nonexistent): expected false")
	}
}

// TestMemoryEngine_VersionedStorage_Property uses rapid to generate random
// sequences of MapSet/MapDelete operations on multiple keys, then verifies
// that MapGetAsOf returns the correct value for every key at every recorded
// timestamp.
func TestMemoryEngine_VersionedStorage_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		eng := NewMemoryEngineWithVersioning()
		defer eng.Close()

		ctx := context.Background()
		mb := eng.(MapBackend)
		vs := eng.(VersionedStorage)

		keys := []string{"a", "b", "c"}

		type stateSnapshot struct {
			ts    time.Time
			state map[string]int64 // key → value (absent = deleted/not-set)
		}

		var timeline []stateSnapshot

		current := map[string]int64{}

		numOps := rapid.IntRange(5, 25).Draw(rt, "numOps")

		for range numOps {
			key := rapid.SampledFrom(keys).Draw(rt, "key")

			_, hasKey := current[key]

			if hasKey && rapid.Bool().Draw(rt, "delete") {
				if err := mb.MapDelete(ctx, "items", key); err != nil {
					rt.Fatalf("MapDelete: %v", err)
				}

				delete(current, key)
			} else {
				val := rapid.Int64Range(0, 1000).Draw(rt, "val")
				if err := mb.MapSet(ctx, "items", key, val); err != nil {
					rt.Fatalf("MapSet: %v", err)
				}

				current[key] = val
			}

			// Capture timestamp AFTER the write so engine_ts <= ts.
			ts := time.Now()

			snap := stateSnapshot{ts: ts, state: make(map[string]int64, len(current))}
			for k, v := range current {
				snap.state[k] = v
			}

			timeline = append(timeline, snap)

			time.Sleep(time.Microsecond) // ensure distinct timestamps
		}

		// Verify: at each timestamp, every key matches the reference model.
		for _, snap := range timeline {
			for _, key := range keys {
				val, err := vs.MapGetAsOf(ctx, "items", key, snap.ts)
				expected, existed := snap.state[key]

				if !existed {
					if !errors.Is(err, ErrNotFound) {
						rt.Fatalf("AsOf(%s, %v): expected ErrNotFound, got val=%v err=%v",
							key, snap.ts, val, err)
					}

					continue
				}

				if err != nil {
					rt.Fatalf("AsOf(%s, %v): unexpected error %v", key, snap.ts, err)
				}

				got, ok := val.(int64)
				if !ok {
					rt.Fatalf("AsOf(%s, %v): expected int64, got %T (%v)", key, snap.ts, val, val)
				}

				if got != expected {
					rt.Fatalf("AsOf(%s, %v): got %d, want %d", key, snap.ts, got, expected)
				}
			}
		}

		// Before the first write: every key should return ErrNotFound.
		if len(timeline) > 0 {
			before := timeline[0].ts.Add(-time.Hour)

			for _, key := range keys {
				_, err := vs.MapGetAsOf(ctx, "items", key, before)
				if !errors.Is(err, ErrNotFound) {
					rt.Fatalf("AsOf(%s, before-first): expected ErrNotFound, got %v", key, err)
				}
			}
		}
	})
}
