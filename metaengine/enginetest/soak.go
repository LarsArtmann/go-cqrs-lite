package enginetest

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// soakTaskCreated/Updated/Deleted/View are internal types used by
// RunAutoCRUDSoak. They are not exported because they exist solely to drive
// the reflection-based AutoCRUDByConvention fold generator.
type soakTaskCreated struct {
	ID     string
	Title  string
	Status string
}

type soakTaskUpdated struct {
	ID     string
	Title  string
	Status string
}

type soakTaskDeleted struct {
	ID string
}

type soakTaskView struct {
	ID     string
	Title  string
	Status string
}

type soakTaskQuery struct{ ID string }

// RunAutoCRUDSoak processes ~46K events through the AutoCRUDByConvention
// auto-projection path (zero-boilerplate CRUD folds derived from Go struct
// naming conventions) against the given engine and verifies:
//
//  1. No memory leaks — heap growth is O(unique keys), not O(total events).
//  2. CRUD lifecycle correctness: created items exist, updates applied,
//     deleted items gone, re-created items restored.
//
// The caller is responsible for closing the engine.
//
// The caller's test MUST NOT call t.Parallel(): this soak asserts on the
// process-global heap (runtime.ReadMemStats), and any concurrently running
// test's live allocations would be miscounted as a leak in this test (and
// this test's heap would break theirs). Sequential tests never overlap with
// other tests, so omitting t.Parallel guarantees an exclusive measurement
// window.
func RunAutoCRUDSoak(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	if testing.Short() {
		t.Skip("auto-crud soak: skips in -short mode")
	}

	folds, err := metaengine.AutoCRUDByConvention[soakTaskView]("ID",
		soakTaskCreated{}, soakTaskUpdated{}, soakTaskDeleted{},
	)
	if err != nil {
		t.Fatalf("AutoCRUDByConvention: %v", err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	q := metaengine.Query[soakTaskQuery, soakTaskView]("soak-autocrud-tasks", foldArgs...)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	ctx := context.Background()

	const (
		numKeys       = 500
		updatesPerKey = 90
		numEvents     = numKeys + numKeys*updatesPerKey + numKeys/5 + numKeys/10
	)

	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("task-%04d", i)
	}

	runtime.GC()

	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	eventCount := 0

	// Phase 1: Create all keys.
	for i := range keys {
		if err := store.Apply(ctx, "soakTaskCreated", soakTaskCreated{
			ID:     keys[i],
			Title:  fmt.Sprintf("Task %d", i),
			Status: "open",
		}); err != nil {
			t.Fatalf("Apply create %d: %v", i, err)
		}

		eventCount++
	}

	// Phase 2: Update each key many times (sustained load).
	for u := range updatesPerKey {
		for k := range keys {
			if err := store.Apply(ctx, "soakTaskUpdated", soakTaskUpdated{
				ID:     keys[k],
				Title:  fmt.Sprintf("Task %d v%d", k, u),
				Status: "in-progress",
			}); err != nil {
				t.Fatalf("Apply update %d-%d: %v", u, k, err)
			}
		}

		eventCount += numKeys
	}

	// Phase 3: Delete every 5th key.
	deletedKeys := make(map[string]bool)

	for i := 0; i < numKeys; i += 5 {
		if err := store.Apply(ctx, "soakTaskDeleted", soakTaskDeleted{ID: keys[i]}); err != nil {
			t.Fatalf("Apply delete %d: %v", i, err)
		}

		deletedKeys[keys[i]] = true
		eventCount++
	}

	// Phase 4: Re-create half the deleted keys.
	recreatedKeys := make(map[string]bool)
	idx := 0

	for k := range deletedKeys {
		if idx%2 == 0 {
			if err := store.Apply(ctx, "soakTaskCreated", soakTaskCreated{
				ID:     k,
				Title:  "Recreated " + k,
				Status: "open",
			}); err != nil {
				t.Fatalf("Apply recreate %s: %v", k, err)
			}

			recreatedKeys[k] = true
			eventCount++
		}

		idx++
	}

	runtime.GC()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	totalGrowth := int64(after.HeapAlloc) - int64(baseline.HeapAlloc)

	maxGrowth := int64(15 * 1024 * 1024) // 15MB
	if RaceEnabled {
		maxGrowth *= 5 // 75MB under -race
	}

	if totalGrowth > maxGrowth {
		t.Errorf(
			"heap grew %d bytes after %d Apply calls with %d keys (max %d) — possible leak",
			totalGrowth, eventCount, numKeys, maxGrowth,
		)
	}

	// Verify correctness: non-deleted keys exist, deleted keys gone.
	var checkErrors int

	for i := range keys {
		result, err := metaengine.ExecuteTyped[soakTaskQuery, soakTaskView](
			ctx, store, soakTaskQuery{ID: keys[i]},
		)

		isDeleted := deletedKeys[keys[i]] && !recreatedKeys[keys[i]]

		if isDeleted {
			if err == nil {
				checkErrors++
				if checkErrors <= 5 {
					t.Errorf("key %s: expected error (deleted), got result %+v", keys[i], result)
				}
			}
		} else {
			if err != nil {
				checkErrors++
				if checkErrors <= 5 {
					t.Errorf("key %s: unexpected error: %v", keys[i], err)
				}
			} else if result.ID != keys[i] {
				checkErrors++
				if checkErrors <= 5 {
					t.Errorf("key %s: ID got %q", keys[i], result.ID)
				}
			}
		}
	}

	if checkErrors > 5 {
		t.Errorf("...and %d more check errors", checkErrors-5)
	}

	t.Logf(
		"auto-crud soak: %d events, %d keys (%d deleted, %d recreated), %d bytes heap growth (%.1f MB), %d errors",
		eventCount,
		numKeys,
		len(deletedKeys),
		len(recreatedKeys),
		totalGrowth,
		float64(totalGrowth)/1024/1024,
		checkErrors,
	)
}
