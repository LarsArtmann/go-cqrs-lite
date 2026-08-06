package metaengine

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

)

// --- P0-1: IN filter silent-drop fix ---

func TestINFilter_PushdownPath(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "B", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "C", Status: "open"}},
	})

	reader := NewReader[testTask](store, "tasks")

	// WithIn on pushdown path (sqliteEngine has RawScanReader)
	results, err := reader.Scan(ctx, WithIn("Status", []any{"open"}))
	if err != nil {
		t.Fatalf("Scan with IN filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("IN filter on pushdown: expected 2, got %d", len(results))
	}
}

// --- P0-3: ErrNotFound ---

func TestErrNotFound_ExecuteTyped(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	_, err := ExecuteTyped[testFindTask, testTask](
		context.Background(),
		store,
		testFindTask{ID: "missing"},
	)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- P0-4: ErrLayoutConflict ---

func TestErrLayoutConflict(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	lp := eng.(LayoutPlanner)
	_ = lp.ApplyLayout("test_col", []string{"a", "b"}, []string{"c"})
	err := lp.ApplyLayout("test_col", []string{"x", "y"}, []string{"z"})
	if !errors.Is(err, ErrLayoutConflict) {
		t.Errorf("expected ErrLayoutConflict, got %v", err)
	}

	// Same columns should be idempotent
	err = lp.ApplyLayout("test_col", []string{"a", "b"}, []string{"c"})
	if err != nil {
		t.Errorf("idempotent ApplyLayout failed: %v", err)
	}
}

// --- P1-1: SQL COUNT pushdown ---

func TestCountPushdown(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Status: "open"}},
	})

	reader := NewReader[testTask](store, "tasks")
	count, err := reader.Count(ctx, WithFilter("Status", FilterEq, "open"))
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// --- P1-3: OR filters ---

func TestORFilter(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Status: "pending"}},
	})

	reader := NewReader[testTask](store, "tasks")
	results, err := reader.Scan(ctx, WithOr(
		FilterSpec{Column: "Status", Op: FilterEq, Value: "open"},
		FilterSpec{Column: "Status", Op: FilterEq, Value: "done"},
	))
	if err != nil {
		t.Fatalf("Scan with OR: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("OR filter: expected 2, got %d", len(results))
	}
}

// --- P1-4: Compound sort ---

func TestCompoundSort(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Status: "open", Title: "B"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Status: "open", Title: "A"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Status: "done", Title: "C"}},
	})

	reader := NewReader[testTask](store, "tasks")
	results, err := reader.Scan(ctx, WithSortColumns(
		SortColumn{Column: "Status", Desc: false},
		SortColumn{Column: "Title", Desc: false},
	))
	if err != nil {
		t.Fatalf("Scan with compound sort: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}

	// Sorted by Status then Title: done/C, open/A, open/B
	if results[0].Title != "C" || results[1].Title != "A" || results[2].Title != "B" {
		t.Errorf("compound sort order wrong: %s, %s, %s",
			results[0].Title, results[1].Title, results[2].Title)
	}
}

// --- P1-5: GroupBy ---

func TestGroupBy(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Status: "done"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Status: "open"}},
	})

	reader := NewReader[testTask](store, "tasks")
	groups, err := reader.GroupBy(ctx, "Status")
	if err != nil {
		t.Fatalf("GroupBy: %v", err)
	}

	if len(groups["open"]) != 2 {
		t.Errorf("group 'open': expected 2, got %d", len(groups["open"]))
	}

	if len(groups["done"]) != 1 {
		t.Errorf("group 'done': expected 1, got %d", len(groups["done"]))
	}
}

// --- P1-6: Schema enforcement ---

func TestSchemaEnforcement_NoMismatchForCorrectTypes(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	// Should not have schema mismatch warnings for correct types
	for _, diag := range store.Plan().Diagnostics {
		if strings.Contains(diag.Message, "fold for") &&
			strings.Contains(diag.Message, "result type is") {
			t.Errorf("unexpected schema mismatch: %s", diag.Message)
		}
	}
}

// testWrongResult is a deliberately different type from testTask to trigger
// the schema mismatch diagnostic.
type testWrongResult struct {
	UnexpectedField int
}

func TestSchemaEnforcement_DetectsTypeMismatch(t *testing.T) {
	t.Parallel()

	// Fold returns testTask but query declares testWrongResult as result type.
	q := Query[testFindTask, testWrongResult](
		"schema_mismatch_test",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, diag := range store.Plan().Diagnostics {
		if diag.Level == DiagLevelWarn &&
			strings.Contains(diag.Message, "fold for") &&
			strings.Contains(diag.Message, "testTask") &&
			strings.Contains(diag.Message, "testWrongResult") {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("expected schema mismatch warning but none found among %d diagnostics",
			len(store.Plan().Diagnostics))
		for _, d := range store.Plan().Diagnostics {
			t.Logf("  %s", d)
		}
	}
}

// --- P1-7: Transaction API (real behavior test) ---

func TestTransaction_CommitRollback(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	ctx := context.Background()

	// Verify Transactional interface
	txEng, ok := eng.(Transactional)
	if !ok {
		t.Fatal("expected sqliteEngine to implement Transactional")
	}

	// Commit path: write inside tx, verify visible after commit
	err := txEng.RunInTx(ctx, func(ctx context.Context) error {
		mb := eng.(MapBackend)

		return mb.MapSet(ctx, "col1", "key1", "value1")
	})
	if err != nil {
		t.Fatalf("RunInTx commit: %v", err)
	}

	mb := eng.(MapBackend)
	val, found, err := mb.MapGet(ctx, "col1", "key1")
	if err != nil || !found || val != "value1" {
		t.Errorf("after commit: expected value1, got val=%v found=%v err=%v", val, found, err)
	}

	// Rollback path: write inside tx, return error → data must NOT persist
	err = txEng.RunInTx(ctx, func(ctx context.Context) error {
		mb := eng.(MapBackend)
		if err := mb.MapSet(ctx, "col1", "key2", "value2"); err != nil {
			return err
		}

		return errors.New("intentional rollback")
	})
	if err == nil {
		t.Fatal("expected error from RunInTx rollback")
	}

	_, found, _ = mb.MapGet(ctx, "col1", "key2")
	if found {
		t.Error("after rollback: key2 should NOT exist")
	}

	// Original data must still be intact
	val, found, _ = mb.MapGet(ctx, "col1", "key1")
	if !found || val != "value1" {
		t.Error("after rollback: key1 should still exist")
	}
}


func _skipped_sqlite_test_0(t *testing.T) {
	t.Skip("requires SQLite engine — moved to sqliteengine module after ADR-0115")
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Atomic batch: both events succeed → both visible
	err = store.InTransaction(ctx, func(ctx context.Context) error {
		if err := store.Apply(ctx, "task_created", testTask{ID: "t1"}); err != nil {
			return err
		}

		return store.Apply(ctx, "task_created", testTask{ID: "t2"})
	})
	if err != nil {
		t.Fatalf("InTransaction commit: %v", err)
	}

	r1, err := ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})
	if err != nil {
		t.Errorf("t1 should exist after commit: %v", err)
	}
	if r1.ID != "t1" {
		t.Errorf("expected t1, got %s", r1.ID)
	}

	// Failed batch: second event fails → first must rollback
	err = store.InTransaction(ctx, func(ctx context.Context) error {
		if err := store.Apply(ctx, "task_created", testTask{ID: "t3"}); err != nil {
			return err
		}

		return errors.New("deliberate failure")
	})
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t3"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("t3 should NOT exist after rollback: %v", err)
	}
}


func _skipped_sqlite_test_1(t *testing.T) {
	t.Skip("requires SQLite engine — moved to sqliteengine module after ADR-0115")
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	ctx := context.Background()

	mb := eng.(MapBackend)
	_ = mb.MapSet(ctx, "col", "k", 10)

	// MapUpdate inside a transaction must work without nested BeginTx
	txEng := eng.(Transactional)
	err := txEng.RunInTx(ctx, func(ctx context.Context) error {
		mu := eng.(MapUpdater)

		return mu.MapUpdate(ctx, "col", "k", func(prev any) any {
			return prev.(float64) + 5
		})
	})
	if err != nil {
		t.Fatalf("MapUpdate in tx: %v", err)
	}

	val, _, _ := mb.MapGet(ctx, "col", "k")
	if val != float64(15) {
		t.Errorf("expected 15, got %v", val)
	}
}

func TestTransactionInterface(t *testing.T) {
	t.Parallel()

	// Memory engine (no Transactional) — fn runs directly without tx
	memStore, _ := Plan([]Engine{NewMemoryEngine()}, testTaskQuery())
	ctx := context.Background()
	err := memStore.InTransaction(ctx, func(ctx context.Context) error {
		return memStore.Apply(ctx, "task_created", testTask{ID: "t1"})
	})
	if err != nil {
		t.Errorf("InTransaction on memory engine: %v", err)
	}
}

// --- P3-6: Plan visualization ---

func TestDotGraph(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	graph := store.Plan().DotGraph()
	if !strings.Contains(graph, "digraph") {
		t.Error("expected 'digraph' in DotGraph output")
	}
}

// --- P3-7: Cost accuracy reporter ---

func TestCostAccuracyReporter(t *testing.T) {
	t.Parallel()

	reporter := NewCostAccuracyReporter(10)
	reporter.Record("test_query", 5*time.Millisecond)

	// Should not panic on nil plan
	reports := reporter.Report(&PlanResult{})
	_ = reports // no queries, no reports
}

// --- P2-4: Checksums ---

func TestChecksum(t *testing.T) {
	data := []byte("hello world")
	cs := Checksum(data)
	if !VerifyChecksum(data, cs) {
		t.Error("checksum verification failed")
	}

	if VerifyChecksum([]byte("tampered"), cs) {
		t.Error("checksum should fail for tampered data")
	}
}

// --- P5-4: Export/Import ---

func TestExportImport(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx := context.Background()
	_ = store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "Task 1"}},
	})

	var buf bytes.Buffer
	if err := store.Export(ctx, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	if !strings.Contains(buf.String(), "tasks") {
		t.Error("expected 'tasks' in export")
	}
}

// --- P3-1: Cost calibration ---

func TestCalibrateEngine(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	// Before calibration: NsPerRead/NsPerWrite are zero (only NsPerOp is set).
	before := eng.Profile()
	if before.NsPerRead != 0 || before.NsPerWrite != 0 {
		t.Fatalf("precondition: expected zero NsPerRead/NsPerWrite, got read=%f write=%f",
			before.NsPerRead, before.NsPerWrite)
	}

	CalibrateEngine(eng, 100)

	// After calibration: NsPerRead/NsPerWrite must be non-zero — proves
	// the calibration values actually reached the engine's profile, not a
	// discarded value copy (the bug this test guards against).
	after := eng.Profile()

	if after.NsPerRead <= 0 {
		t.Errorf("expected calibrated NsPerRead > 0, got %f", after.NsPerRead)
	}

	if after.NsPerWrite <= 0 {
		t.Errorf("expected calibrated NsPerWrite > 0, got %f", after.NsPerWrite)
	}

	if after.NsPerOp <= 0 {
		t.Errorf("expected calibrated NsPerOp > 0, got %f", after.NsPerOp)
	}
}

// --- P2-1: Read coalescer ---

func TestReadCoalescer(t *testing.T) {
	t.Parallel()

	rc := NewReadCoalescer()
	callCount := 0

	v1, err := rc.Do("key1", func() (any, error) {
		callCount++

		return "result", nil
	})
	if err != nil || v1 != "result" {
		t.Errorf("unexpected result: %v, %v", v1, err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// --- Hooks (P3-2/3/4/5) ---

func TestHooks(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	var foldCount, execCount int
	var foldErr error
	WithHooks(store, Hooks{
		OnFold: func(col, evt string, kind FoldKind, d time.Duration, err error) {
			foldCount++
			foldErr = err
		},
		OnExecute: func(col string, pattern ReadPattern, d time.Duration, err error) {
			execCount++
		},
	})

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1"})

	if foldCount != 1 {
		t.Errorf("expected 1 fold hook call, got %d", foldCount)
	}

	if foldErr != nil {
		t.Errorf("expected nil fold error on success, got %v", foldErr)
	}

	_, _ = ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})

	if execCount != 1 {
		t.Errorf("expected 1 execute hook call, got %d", execCount)
	}
}
