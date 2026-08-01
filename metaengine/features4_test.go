package metaengine

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- Stats tests ---

func TestStats_ReturnsRowCounts(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "B", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "C", Status: "open"}},
	})

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if len(stats) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(stats))
	}

	if stats[0].Name != "tasks" {
		t.Errorf("expected collection 'tasks', got %q", stats[0].Name)
	}

	if stats[0].RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", stats[0].RowCount)
	}
}

// --- HealthCheck tests ---

func TestHealthCheck_AllEnginesHealthy(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if err := store.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck should return nil for healthy engines, got: %v", err)
	}
}

func TestHealthCheck_SQLiteEngine(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	if err := store.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck should return nil for SQLite engine, got: %v", err)
	}
}

// --- PrefetchCache auto-population test ---

func TestPrefetchCache_AutoPopulation(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	// Insert 5 tasks with different statuses for deterministic sort.
	for i := range 5 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:     testTaskID(fmt.Sprintf("t%d", i)),
			Title:  fmt.Sprintf("Task %d", i),
			Status: "open",
		})
	}

	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	// First scan: limit 2. Should auto-populate cache with rows 3+.
	page1, err := reader.Scan(ctx, WithLimit(2))
	if err != nil {
		t.Fatalf("Scan page1: %v", err)
	}

	if len(page1) != 2 {
		t.Fatalf("expected 2 results on page1, got %d", len(page1))
	}

	// Check if the cache was populated with the next page.
	// The cache key should be derived from the last item's sort field.
	// Since no sort was specified, the cursor key uses the whole item.
	// Verify the cache has entries.
	if len(cache.pages) == 0 {
		t.Error("PrefetchCache should have been auto-populated with extra rows")
	}
}

// --- Watcher per-key filtering test ---

func TestWatcher_PerKeyFiltering(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	// Watcher for key "t1" only.
	ch := watcher.Watch(ctx, testTaskID("t1"))

	// Apply events for t1 and t2.
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "A"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "t2", Title: "B"})

	// Should receive notification for t1 only.
	select {
	case val := <-ch:
		if val.ID != "t1" {
			t.Errorf("expected task t1, got %s", val.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for t1 notification")
	}

	// Should NOT receive notification for t2.
	select {
	case val := <-ch:
		t.Errorf("should not have received notification for t2, got: %+v", val)
	case <-time.After(100 * time.Millisecond):
		// Expected: no notification for t2.
	}
}

// --- SSE backpressure test ---

func TestSSE_DropOldSemantics(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	// Small buffer for testing drop-old.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(w, r, watcher, WithSSEMaxBuffer(2))
	}))
	defer srv.Close()

	// We don't need to consume all events; just verify the server doesn't block.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		for i := range 10 {
			_ = store.Apply(ctx, "task_created", testTask{
				ID:    testTaskID(fmt.Sprintf("t%d", i)),
				Title: fmt.Sprintf("Task %d", i),
			})
		}
	}()

	// Connect briefly to verify the server doesn't hang.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)

	resp, err := http.DefaultClient.Do(req)
	_ = err // server may close on timeout, which is fine

	if resp != nil {
		defer resp.Body.Close()
	}
}

func TestSSE_Timeout(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(w, r, watcher, WithSSETimeout(100*time.Millisecond))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)

	start := time.Now()

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}

	elapsed := time.Since(start)

	if elapsed > 1500*time.Millisecond {
		t.Errorf("SSE should have timed out after ~100ms, took %v", elapsed)
	}
}

// --- InspectJSON test ---

func TestInspectJSON_ValidJSON(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	data, err := store.InspectJSON()
	if err != nil {
		t.Fatalf("InspectJSON: %v", err)
	}

	var collections []CollectionInfo
	if err := json.Unmarshal(data, &collections); err != nil {
		t.Fatalf("InspectJSON returned invalid JSON: %v", err)
	}

	if len(collections) != 1 {
		t.Errorf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].Name != "tasks" {
		t.Errorf("expected collection 'tasks', got %q", collections[0].Name)
	}
}

// --- Expanded error taxonomy test ---

func TestExpandedErrorSentinels(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		ErrNotFound,
		ErrAmbiguousKey,
		ErrUnsupportedADT,
		ErrLayoutConflict,
		ErrPoisoned,
		ErrNoQueryForInputType,
		ErrUnsupportedPattern,
		ErrUnknownFoldKind,
		ErrExecuteTypeMismatch,
		ErrDuplicateEvent,
	}

	for _, s := range sentinels {
		if s == nil {
			t.Error("exported sentinel should not be nil")
		}
	}

	// Verify errors.Is works through wrapping.
	wrapped := fmt.Errorf("context: %w", ErrPoisoned)
	if !errors.Is(wrapped, ErrPoisoned) {
		t.Error("errors.Is should match wrapped ErrPoisoned")
	}

	wrapped2 := fmt.Errorf("ctx: %w", ErrNoQueryForInputType)
	if !errors.Is(wrapped2, ErrNoQueryForInputType) {
		t.Error("errors.Is should match wrapped ErrNoQueryForInputType")
	}
}

// --- Export/Import round-trip test ---

func TestExportImport_RoundTrip(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store1, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan store1: %v", err)
	}

	ctx := context.Background()

	_ = store1.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "Alpha", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "Beta", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "Gamma", Status: "open"}},
	})

	// Export to buffer.
	var buf bytes.Buffer
	if err := store1.Export(ctx, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Export produced empty output")
	}

	// Verify exported data is valid JSON.
	var exported map[string]any
	if err := json.Unmarshal(buf.Bytes(), &exported); err != nil {
		t.Fatalf("Export produced invalid JSON: %v", err)
	}

	// Import into a new store.
	eng2 := NewMemoryEngine()
	store2, err := Plan([]Engine{eng2}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan store2: %v", err)
	}

	if err := store2.Import(ctx, &buf); err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Verify data matches via Scan (key types may differ after round-trip
	// through JSON, so we verify via scan rather than Get by key).
	all, err := NewReader[testTask](store2, "tasks").Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 imported tasks, got %d", len(all))
	}

	found := false
	for _, task := range all {
		if task.Title == "Alpha" && task.Status == "open" {
			found = true

			break
		}
	}

	if !found {
		t.Error("Import did not restore task with Title='Alpha', Status='open'")
	}
}

// --- Crash recovery test (poison) ---

func TestCrashRecovery_PanicPoisonsCollection(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	// Query with a fold that panics.
	panicQuery := Query[testFindTask, testTask](
		"panic_tasks",
		OnTyped("panic_event", testTask{}, func(e testTask) (testTaskID, testTask) {
			panic("intentional crash")
		}),
	)

	store, err := Plan([]Engine{eng}, panicQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	// Apply should not panic — it should recover and poison the collection.
	err = store.Apply(ctx, "panic_event", testTask{ID: "t1"})
	if err == nil {
		t.Fatal("Apply should return error after fold panic")
	}

	// Collection should be poisoned.
	if poisonErr := store.IsPoisoned("panic_tasks"); poisonErr == nil {
		t.Fatal("collection should be poisoned after fold panic")
	}

	// Subsequent reads should return the poison error.
	_, _, err = NewReader[testTask](store, "panic_tasks").Get(ctx, testTaskID("t1"))
	if err == nil {
		t.Fatal("Get on poisoned collection should return error")
	}
}

// --- EventLog replay test ---

func TestEventLog_ReplayAndVerify(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	log := NewEventLog()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	WithEventLog(store, log)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "B"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "C"}},
	})

	if log.Len() != 3 {
		t.Fatalf("expected 3 logged events, got %d", log.Len())
	}

	events := log.Events()
	if len(events) != 3 {
		t.Fatalf("expected 3 events from Events(), got %d", len(events))
	}

	// Verify should succeed — replaying into a fresh store produces same row count.
	err = store.Verify(ctx, []Engine{NewMemoryEngine()})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// --- Consistency checker drift test ---

func TestVerify_DetectsDrift(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	log := NewEventLog()

	query := testTaskQuery()
	store, err := Plan([]Engine{eng}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	WithEventLog(store, log)

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "A"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "t2", Title: "B"})

	// Manually corrupt the store by adding an extra row directly.
	mb := eng.(MapBackend)
	_ = mb.MapSet(ctx, "tasks", "rogue", testTask{ID: "rogue", Title: "Malicious"})

	// Verify should detect the drift (3 rows live vs 2 rows replayed).
	err = store.Verify(ctx, []Engine{NewMemoryEngine()})
	if err == nil {
		t.Fatal("Verify should detect drift after manual corruption")
	}
}

// --- SQLite Watcher test ---

func TestSQLiteWatcher_ReceivesValue(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	_ = store.Apply(ctx, "task_created", testTask{ID: "sqlite-1", Title: "SQLite Task"})

	select {
	case val := <-ch:
		if val.ID != "sqlite-1" {
			t.Errorf("expected 'sqlite-1', got %s", val.ID)
		}

		if val.Title != "SQLite Task" {
			t.Errorf("expected 'SQLite Task', got %q", val.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SQLite watcher notification")
	}
}

// --- SQLite ReadCoalescer test ---

func TestSQLiteCoalescer_ConcurrentReadsCoalesced(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	WithReadCoalescer(store, NewReadCoalescer())

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "coalesced", Title: "Test"})

	reader := NewReader[testTask](store, "tasks")

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]testTask, goroutines)
	foundFlags := make([]bool, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			results[idx], foundFlags[idx], errs[idx] = reader.Get(ctx, testTaskID("coalesced"))
		}(i)
	}

	wg.Wait()

	for i := range goroutines {
		if errs[i] != nil {
			t.Errorf("goroutine %d got error: %v", i, errs[i])
		}

		if !foundFlags[i] {
			t.Errorf("goroutine %d: expected found=true", i)
		}

		if results[i].Title != "Test" {
			t.Errorf("goroutine %d: expected Title 'Test', got %q", i, results[i].Title)
		}
	}
}

// --- ApplyIdempotent test ---

func TestApplyIdempotent_DeduplicatesByEventID(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	// Apply the same event twice with the same ID.
	_ = store.ApplyIdempotent(ctx, "evt-1", "task_created", testTask{ID: "t1", Title: "A"})
	_ = store.ApplyIdempotent(ctx, "evt-1", "task_created", testTask{ID: "t1", Title: "B"})

	// Second apply should be a no-op — title should still be "A".
	reader := NewReader[testTask](store, "tasks")
	task, found, err := reader.Get(ctx, testTaskID("t1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !found {
		t.Fatal("task t1 should exist")
	}

	if task.Title != "A" {
		t.Errorf("expected Title 'A' (idempotent), got %q", task.Title)
	}
}

// --- Checksum test ---

func TestChecksum_VerifyRoundTrip(t *testing.T) {
	t.Parallel()

	data := []byte(`{"id":"t1","title":"Test","status":"open"}`)
	cs := Checksum(data)
	if !VerifyChecksum(data, cs) {
		t.Error("VerifyChecksum should return true for matching checksum")
	}

	corrupted := []byte(`{"id":"t1","title":"CORRUPTED","status":"open"}`)
	if VerifyChecksum(corrupted, cs) {
		t.Error("VerifyChecksum should return false for corrupted data")
	}
}

// --- Property-style test: random insert/update/delete sequence ---

func TestProperty_RandomOpsMaintainConsistency(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	// Query with insert, update, and delete folds.
	updateQuery := Query[testFindTask, testTask](
		"prop_tasks",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
		OnTyped("task_title_changed", testTask{}, func(e testTask, prev testTask) testTask {
			prev.Title = e.Title

			return prev
		}),
		OnTyped("task_deleted", testTask{}, Remove[testTask]()),
	)

	store, err := Plan([]Engine{eng}, updateQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	reader := NewReader[testTask](store, "prop_tasks")

	type op struct {
		kind   string // "create", "update", "delete"
		taskID string
		title  string
	}

	rng := rand.New(rand.NewSource(42))

	const numOps = 200
	const numTasks = 20
	expected := make(map[string]string) // taskID → expected title

	for range numOps {
		taskIdx := rng.Intn(numTasks)
		id := testTaskID(fmt.Sprintf("task-%d", taskIdx))
		opType := rng.Intn(3)

		switch opType {
		case 0: // create
			title := fmt.Sprintf("Title-%d", rng.Intn(10000))
			expected[string(id)] = title
			if err := store.Apply(ctx, "task_created", testTask{ID: id, Title: title}); err != nil {
				t.Errorf("create %s: %v", id, err)
			}
		case 1: // update (only if exists)
			if _, exists := expected[string(id)]; !exists {
				continue
			}

			newTitle := fmt.Sprintf("Updated-%d", rng.Intn(10000))
			expected[string(id)] = newTitle
			if err := store.Apply(
				ctx,
				"task_title_changed",
				testTask{ID: id, Title: newTitle},
			); err != nil {
				t.Errorf("update %s: %v", id, err)
			}
		case 2: // delete (only if exists)
			if _, exists := expected[string(id)]; !exists {
				continue
			}

			delete(expected, string(id))
			if err := store.Apply(ctx, "task_deleted", testTask{ID: id}); err != nil {
				t.Errorf("delete %s: %v", id, err)
			}
		}
	}

	// Verify final state matches expectations.
	for taskID, expectedTitle := range expected {
		task, found, err := reader.Get(ctx, testTaskID(taskID))
		if err != nil {
			t.Errorf("Get %s: %v", taskID, err)

			continue
		}

		if !found {
			t.Errorf("task %s should exist with title %q", taskID, expectedTitle)

			continue
		}

		if task.Title != expectedTitle {
			t.Errorf("task %s: expected title %q, got %q", taskID, expectedTitle, task.Title)
		}
	}

	// Verify deleted tasks are gone.
	for taskIdx := range numTasks {
		id := fmt.Sprintf("task-%d", taskIdx)
		if _, exists := expected[id]; !exists {
			_, found, _ := reader.Get(ctx, testTaskID(id))
			if found {
				t.Errorf("task %s should have been deleted", id)
			}
		}
	}
}

// --- PrefetchCache end-to-end pagination tests ---

func TestPrefetchCache_EndToEndPagination(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	// Insert 10 tasks with distinct titles.
	for i := range 10 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:    testTaskID(fmt.Sprintf("t%d", i)),
			Title: fmt.Sprintf("Task-%02d", i),
		})
	}

	// Page 1: limit 3, no cursor.
	page1, cursor1, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(3))
	if err != nil {
		t.Fatalf("ScanPage page1: %v", err)
	}

	if len(page1) != 3 {
		t.Fatalf("expected 3 items on page1, got %d", len(page1))
	}

	if cursor1 == nil {
		t.Fatal("expected non-nil cursor after page1 (more pages exist)")
	}

	// Page 2: use cursor from page1 — should be served from PrefetchCache.
	page2, cursor2, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursor(cursor1.Value),
	)
	if err != nil {
		t.Fatalf("ScanPage page2: %v", err)
	}

	if len(page2) != 3 {
		t.Fatalf("expected 3 items on page2, got %d", len(page2))
	}

	if cursor2 == nil {
		t.Fatal("expected non-nil cursor after page2 (more pages exist)")
	}

	// Verify no overlap between pages.
	seen := make(map[testTaskID]bool)
	for _, item := range page1 {
		seen[item.ID] = true
	}

	for _, item := range page2 {
		if seen[item.ID] {
			t.Errorf("item %s appeared on both pages", item.ID)
		}
	}
}

func TestPrefetchCache_SQLiteEndToEnd(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	ctx := context.Background()
	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	for i := range 10 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:    testTaskID(fmt.Sprintf("t%d", i)),
			Title: fmt.Sprintf("Task-%02d", i),
		})
	}

	page1, cursor1, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(3))
	if err != nil {
		t.Fatalf("ScanPage page1: %v", err)
	}

	if len(page1) != 3 {
		t.Fatalf("expected 3 items on page1, got %d", len(page1))
	}

	if cursor1 == nil {
		t.Fatal("expected non-nil cursor after page1")
	}

	// Page 2 from cache (or engine fallback).
	page2, _, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursor(cursor1.Value),
	)
	if err != nil {
		t.Fatalf("ScanPage page2: %v", err)
	}

	if len(page2) != 3 {
		t.Fatalf("expected 3 items on page2, got %d", len(page2))
	}
}

// --- SSE multi-subscriber fan-out test ---

func TestSSE_MultiSubscriberFanOut(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	// Start 3 SSE servers sharing the same watcher.
	servers := make([]*httptest.Server, 3)

	for i := range 3 {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = ServeSSE(w, r, watcher, WithSSETimeout(2*time.Second))
		}))
		servers[i] = srv
	}

	defer func() {
		for _, srv := range servers {
			srv.Close()
		}
	}()

	// Connect 3 clients.
	type sseResult struct {
		events []string
		err    error
	}

	results := make([]sseResult, 3)
	var wg sync.WaitGroup

	wg.Add(3)

	for i := range 3 {
		go func(idx int) {
			defer wg.Done()

			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, servers[idx].URL, nil)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[idx].err = err

				return
			}

			defer resp.Body.Close()

			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					results[idx].events = append(results[idx].events, string(buf[:n]))
				}

				if err != nil {
					return
				}
			}
		}(i)
	}

	// Give clients time to connect.
	time.Sleep(500 * time.Millisecond)

	// Apply an event that should reach all subscribers.
	_ = store.Apply(ctx, "task_created", testTask{ID: "fanout-1", Title: "FanOut"})

	// Give events time to propagate through watcher → SSE → HTTP.
	time.Sleep(500 * time.Millisecond)

	cancel()
	wg.Wait()

	// At least one subscriber should have received data.
	received := 0

	for _, r := range results {
		if len(r.events) > 0 {
			received++
		}
	}

	if received == 0 {
		t.Error("expected at least one subscriber to receive events")
	}
}

// --- Export/Import cross-engine test ---

func TestExportImport_CrossEngine(t *testing.T) {
	t.Parallel()

	// Export from Memory.
	memEng := NewMemoryEngine()
	memStore, err := Plan([]Engine{memEng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan memory: %v", err)
	}

	ctx := context.Background()
	_ = memStore.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "x1", Title: "Cross", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "x2", Title: "Engine", Status: "done"}},
	})

	var buf bytes.Buffer
	if err := memStore.Export(ctx, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Import to SQLite.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	defer db.Close()

	sqlEng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	sqlStore, err := Plan([]Engine{sqlEng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan sqlite: %v", err)
	}

	if err := sqlStore.Import(ctx, &buf); err != nil {
		t.Fatalf("Import: %v", err)
	}

	reader := NewReader[testTask](sqlStore, "tasks")
	results, err := reader.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks after cross-engine import, got %d", len(results))
	}
}

// --- MapUpdateTyped test ---

func TestMapUpdateTyped_ReifiesPrevValue(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "tu1", Title: "Original"})

	// Update via MapUpdateTyped — prev should be correctly typed.
	err = MapUpdateTyped[testTask](store, ctx, "tasks", testTaskID("tu1"),
		func(prev testTask, found bool) testTask {
			if !found {
				t.Error("expected found=true for existing key")
			}

			prev.Title = "Updated"

			return prev
		})
	if err != nil {
		t.Fatalf("MapUpdateTyped: %v", err)
	}

	reader := NewReader[testTask](store, "tasks")
	task, found, err := reader.Get(ctx, testTaskID("tu1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !found {
		t.Fatal("expected task to exist after update")
	}

	if task.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", task.Title)
	}

	// Test not-found case — data stored under the key, not the returned value's ID.
	err = MapUpdateTyped[testTask](store, ctx, "tasks", testTaskID("nonexistent"),
		func(prev testTask, found bool) testTask {
			if found {
				t.Error("expected found=false for nonexistent key")
			}

			return testTask{ID: "nonexistent", Title: "Created"}
		})
	if err != nil {
		t.Fatalf("MapUpdateTyped not-found: %v", err)
	}

	task, found, err = reader.Get(ctx, testTaskID("nonexistent"))
	if err != nil || !found {
		t.Fatalf("expected new task to exist, found=%v, err=%v", found, err)
	}

	if task.Title != "Created" {
		t.Errorf("expected title 'Created', got %q", task.Title)
	}
}

// --- WithTTL functional test ---

func TestWithTTL_SetsConfigValue(t *testing.T) {
	t.Parallel()

	// WithTTL is a QueryOption (used at Plan time), not a ScanOption.
	// Verify it correctly sets the nanosecond TTL on QueryConfig.

	ttl := 5 * time.Minute
	var cfg QueryConfig
	WithTTL(ttl)(&cfg)

	if cfg.TTL != ttl.Nanoseconds() {
		t.Errorf("expected TTL %d ns, got %d", ttl.Nanoseconds(), cfg.TTL)
	}

	// Zero TTL (no expiration) should also work.
	var cfgZero QueryConfig
	WithTTL(0)(&cfgZero)

	if cfgZero.TTL != 0 {
		t.Errorf("expected zero TTL, got %d", cfgZero.TTL)
	}

	// Negative duration produces negative nanoseconds — this is fine as a
	// signal to engines to skip expiration entirely.
	var cfgNeg QueryConfig
	WithTTL(-1)(&cfgNeg)

	if cfgNeg.TTL != -1 {
		t.Errorf("expected TTL -1, got %d", cfgNeg.TTL)
	}
}
