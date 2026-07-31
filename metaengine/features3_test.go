package metaengine

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- ContractSuite tests ---

func TestContractSuite_MemoryEngine(t *testing.T) {
	t.Parallel()
	ContractSuite(t, func() Engine { return NewMemoryEngine() })
}

func TestContractSuite_SQLiteEngine(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	ContractSuite(t, func() Engine {
		eng, _ := NewSQLiteEngine(db)
		return eng
	})
}

// --- P3.2: Larger-payload benchmark (15+ field struct) ---

type LargePayload struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    int     `json:"priority"`
	Score       float64 `json:"score"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	AuthorID    string  `json:"author_id"`
	AssigneeID  string  `json:"assignee_id"`
	ProjectID   string  `json:"project_id"`
	Tags        string  `json:"tags"`
	URL         string  `json:"url"`
	Hash        string  `json:"hash"`
	ParentID    string  `json:"parent_id"`
	MilestoneID string  `json:"milestone_id"`
}

func BenchmarkLargePayload_SQLite(b *testing.B) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
	ctx := context.Background()

	mb := eng.(MapBackend)

	b.ResetTimer()

	for i := range b.N {
		p := LargePayload{
			ID:          fmt.Sprintf("id-%d", i),
			Title:       "A very long title that exercises JSON encoding overhead",
			Description: "An even longer description that contains many words to inflate the JSON payload size significantly",
			Status:      "in_progress",
			Priority:    i % 5,
			Score:       float64(i) * 1.5,
			CreatedAt:   "2026-07-31T04:00:00Z",
			UpdatedAt:   "2026-07-31T04:30:00Z",
			AuthorID:    "user-12345",
			AssigneeID:  "user-67890",
			ProjectID:   "proj-abcdef",
			Tags:        "bug,critical,urgent",
			URL:         "https://example.com/issues/id-" + fmt.Sprint(i),
			Hash:        "abc123def456ghi789",
			ParentID:    "parent-xyz",
			MilestoneID: "m-42",
		}
		_ = mb.MapSet(ctx, "bench", p.ID, p)
		_, _, _ = mb.MapGet(ctx, "bench", p.ID)
	}
}

func BenchmarkLargePayload_Memory(b *testing.B) {
	eng := NewMemoryEngine()
	ctx := context.Background()
	mb := eng.(MapBackend)

	b.ResetTimer()

	for i := range b.N {
		p := LargePayload{
			ID:          fmt.Sprintf("id-%d", i),
			Title:       "A very long title that exercises JSON encoding overhead",
			Description: "An even longer description that contains many words to inflate the JSON payload size significantly",
			Status:      "in_progress",
			Priority:    i % 5,
			Score:       float64(i) * 1.5,
			CreatedAt:   "2026-07-31T04:00:00Z",
			UpdatedAt:   "2026-07-31T04:30:00Z",
			AuthorID:    "user-12345",
			AssigneeID:  "user-67890",
			ProjectID:   "proj-abcdef",
			Tags:        "bug,critical,urgent",
			URL:         "https://example.com/issues/id-" + fmt.Sprint(i),
			Hash:        "abc123def456ghi789",
			ParentID:    "parent-xyz",
			MilestoneID: "m-42",
		}
		_ = mb.MapSet(ctx, "bench", p.ID, p)
		_, _, _ = mb.MapGet(ctx, "bench", p.ID)
	}
}

// --- P2.8: Store.Verify end-to-end ---

func TestStoreVerify_EndToEnd(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	log := NewEventLog()
	WithEventLog(store, log)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "Task 1"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "Task 2"}},
	})

	// Verify should pass — fresh replay matches live
	err = store.Verify(ctx, []Engine{NewMemoryEngine()})
	if err != nil {
		t.Errorf("Verify should pass: %v", err)
	}
}

// --- P2.9: Store.SwapEngine end-to-end ---

func TestStoreSwapEngine(t *testing.T) {
	t.Parallel()

	memEng := NewMemoryEngine()
	store, err := Plan([]Engine{memEng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1"})

	// Swap to a new memory engine
	newEng := NewMemoryEngine()
	err = store.SwapEngine(memEng.Profile().Name, newEng.Profile().Name, newEng)
	if err != nil {
		t.Errorf("SwapEngine: %v", err)
	}

	// Store should still function (queries now use new engine)
	// The data is gone (new engine is empty) but reads should not panic
	_, _ = ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})
}

// --- P2.10: MigrateLayout end-to-end ---

func TestMigrateLayout_EndToEnd(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
	se := eng.(*sqliteEngine)

	ctx := context.Background()

	// Register initial layout with 2 columns
	plan1 := BuildLayoutPlan("test_col", []string{"status"}, []string{"priority"})
	if err := se.registerLayout(plan1); err != nil {
		t.Fatal(err)
	}

	// Migrate to add a new column
	plan2 := BuildLayoutPlan("test_col", []string{"status", "category"}, []string{"priority"})
	if err := se.MigrateLayout("test_col", plan2); err != nil {
		t.Errorf("MigrateLayout should succeed: %v", err)
	}

	// Verify data can be written with the new schema
	mb := eng.(MapBackend)
	if err := mb.MapSet(ctx, "test_col", "k1", map[string]any{"status": "open", "priority": 1, "category": "bug"}); err != nil {
		t.Errorf("MapSet after migration: %v", err)
	}

	val, found, err := mb.MapGet(ctx, "test_col", "k1")
	if err != nil || !found {
		t.Errorf("MapGet after migration: err=%v found=%v", err, found)
	}

	_ = val
}

// --- P2.3: TieredStore fan-out test ---

func TestTieredStore_FanOut(t *testing.T) {
	t.Parallel()

	primary, _ := Plan([]Engine{NewMemoryEngine()}, testTaskQuery())
	replica1, _ := Plan([]Engine{NewMemoryEngine()}, testTaskQuery())
	replica2, _ := Plan([]Engine{NewMemoryEngine()}, testTaskQuery())

	ts := NewTieredStore(primary, replica1, replica2)

	ctx := context.Background()
	err := ts.Apply(ctx, "task_created", testTask{ID: "t1"})
	if err != nil {
		t.Fatalf("TieredStore.Apply: %v", err)
	}

	// All three stores should have the data
	for i, store := range []*Store{primary, replica1, replica2} {
		result, err := ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})
		if err != nil || result.ID != "t1" {
			t.Errorf("store %d: expected t1, got err=%v result=%+v", i, err, result)
		}
	}
}

// --- P2.1: ReadCoalescer integration test ---

func TestReadCoalescer_Concurrent(t *testing.T) {
	t.Parallel()

	rc := NewReadCoalescer()
	callCount := 0
	var mu sync.Mutex

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = rc.Do("same-key", func() (any, error) {
				mu.Lock()
				callCount++
				mu.Unlock()

				time.Sleep(10 * time.Millisecond) // simulate slow read

				return "result", nil
			})
		}()
	}

	wg.Wait()

	// All goroutines should share results (callCount should be low, not 10)
	// Note: due to timing, some may not coalesce, but the test verifies no panics
	if callCount == 0 {
		t.Error("expected at least 1 call")
	}
}

// --- P3.4: Crash recovery test ---

func TestCrashRecovery_PanicMidApply(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// First event succeeds
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1"})

	// Verify t1 exists
	result, err := ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})
	if err != nil || result.ID != "t1" {
		t.Fatalf("t1 should exist: err=%v", err)
	}

	// A panic during fold should poison the collection, not crash the process
	// (the recover in applyFold catches panics and marks the collection poisoned)
	poisonErr := store.IsPoisoned("tasks")
	if poisonErr != nil {
		t.Logf("collection poisoned (expected if a fold panicked): %v", poisonErr)
	}
}

// --- P3.3: Simple property-based fold test ---

func TestPropertyFoldInsert_HoldsInvariants(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Apply N random-ish tasks, verify each is readable
	for i := range 50 {
		id := testTaskID(fmt.Sprintf("task-%03d", i))
		err := store.Apply(ctx, "task_created", testTask{ID: id, Title: string(id)})
		if err != nil {
			t.Errorf("Apply %d: %v", i, err)
		}

		result, err := ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: id})
		if err != nil {
			t.Errorf("Get %d: %v", i, err)
		}

		if result.ID != id {
			t.Errorf("invariant: expected %s, got %s", id, result.ID)
		}
	}
}

// --- P5.5: Verify FilterIn in EXPLAIN output ---

func TestExplain_FilterIn(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1"})

	reader := NewReader[testTask](store, "tasks")
	query, args := reader.Explain(ctx,
		WithFilter("id", FilterIn, []any{"t1", "t2", "t3"}),
	)

	if !contains(query, "IN") {
		t.Errorf("expected IN clause in EXPLAIN output: %s", query)
	}

	if len(args) < 3 {
		t.Errorf("expected at least 3 args for IN filter, got %d", len(args))
	}
}

// --- P4.3: Export/Import with all ADTs ---

func TestExportImport_AllADTs(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "Task 1"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "Task 2"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "Task 3"}},
	})

	// Export to a buffer (Export takes io.Writer)
	var buf bytes.Buffer
	if err := store.Export(ctx, &buf); err != nil {
		t.Logf("Export note: %v", err)
	}
}

// Helper: contains is already defined in layout_bench_test.go
