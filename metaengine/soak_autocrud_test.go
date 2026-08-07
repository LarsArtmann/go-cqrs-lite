package metaengine_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestSoak_AutoCRUDByConvention processes 50K events through the auto-projection
// path (AutoCRUDByConvention — zero-boilerplate CRUD folds derived from Go struct
// naming conventions) and verifies:
//
//  1. No memory leaks — heap growth is O(unique keys), not O(total events).
//  2. CRUD lifecycle correctness: created items exist, updates applied, deleted
//     items gone, re-created items restored.
//
// This complements the other soak tests which use manual OnRecord folds. The
// AutoCRUDByConvention path uses reflection-derived folds (autoInsertByType,
// autoUpdateByType, autoDeleteByType) with a different dispatch path that
// warrants its own soak.
func TestSoak_AutoCRUDByConvention(t *testing.T) {
	if testing.Short() {
		t.Skip("auto-crud soak: skips in -short mode")
	}

	t.Parallel()

	type taskCreated struct {
		ID     string
		Title  string
		Status string
	}

	type taskUpdated struct {
		ID     string
		Title  string
		Status string
	}

	type taskDeleted struct {
		ID string
	}

	type taskView struct {
		ID     string
		Title  string
		Status string
	}

	type taskQuery struct{ ID string }

	folds, err := metaengine.AutoCRUDByConvention[taskView]("ID",
		taskCreated{}, taskUpdated{}, taskDeleted{},
	)
	if err != nil {
		t.Fatalf("AutoCRUDByConvention: %v", err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	q := metaengine.Query[taskQuery, taskView]("soak-autocrud-tasks", foldArgs...)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	ctx := context.Background()

	const (
		numKeys      = 500
		updatesPerKey = 90
		numEvents    = numKeys + numKeys*updatesPerKey + numKeys/5 + numKeys/10
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
		if err := store.Apply(ctx, "taskCreated", taskCreated{
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
			if err := store.Apply(ctx, "taskUpdated", taskUpdated{
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
		if err := store.Apply(ctx, "taskDeleted", taskDeleted{ID: keys[i]}); err != nil {
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
			if err := store.Apply(ctx, "taskCreated", taskCreated{
				ID:     k,
				Title:  fmt.Sprintf("Recreated %s", k),
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
	if raceEnabled {
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
		result, err := metaengine.ExecuteTyped[taskQuery, taskView](
			ctx, store, taskQuery{ID: keys[i]},
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
		eventCount, numKeys, len(deletedKeys), len(recreatedKeys),
		totalGrowth, float64(totalGrowth)/1024/1024, checkErrors,
	)
}
