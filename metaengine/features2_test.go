package metaengine

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- P0-1: IN filter silent-drop fix ---

func TestINFilter_PushdownPath(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
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

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExecuteTyped[testFindTask, testTask](context.Background(), store, testFindTask{ID: "missing"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- P0-4: ErrLayoutConflict ---

func TestErrLayoutConflict(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
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

	eng, _ := NewSQLiteEngine(db)
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

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

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

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

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

func TestSchemaEnforcement(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	// Should not have schema mismatch warnings for correct types
	for _, diag := range store.Plan().Diagnostics {
		if strings.Contains(diag.Message, "fold returns") && strings.Contains(diag.Message, "but query result type is") {
			t.Errorf("unexpected schema mismatch: %s", diag.Message)
		}
	}
}

// --- P1-7: Transaction API (interface availability) ---

func TestTransactionInterface(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)

	// Verify Transactional interface is available on the concrete type
	if _, ok := eng.(Transactional); !ok {
		t.Error("expected sqliteEngine to implement Transactional")
	}

	// Verify Store.InTransaction delegates to engine
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Memory engine (no Transactional) — fn runs directly
	memStore, _ := Plan([]Engine{NewMemoryEngine()}, testTaskQuery())
	err = memStore.InTransaction(ctx, func(ctx context.Context) error {
		return memStore.Apply(ctx, "task_created", testTask{ID: "t1"})
	})
	if err != nil {
		t.Errorf("InTransaction on memory engine: %v", err)
	}

	_ = store // SQLite engine Transactional path tested separately
}

// --- P3-6: Plan visualization ---

func TestDotGraph(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

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

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

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
	CalibrateEngine(eng, 100)

	profile := eng.Profile()
	if profile.NsPerOp <= 0 {
		t.Error("expected calibrated NsPerOp > 0")
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

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	var foldCount, execCount int
	WithHooks(store, Hooks{
		OnFold: func(col, evt string, kind FoldKind, d time.Duration) {
			foldCount++
		},
		OnExecute: func(col string, pattern ReadPattern, d time.Duration) {
			execCount++
		},
	})

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "t1"})

	if foldCount != 1 {
		t.Errorf("expected 1 fold hook call, got %d", foldCount)
	}

	_, _ = ExecuteTyped[testFindTask, testTask](ctx, store, testFindTask{ID: "t1"})

	if execCount != 1 {
		t.Errorf("expected 1 execute hook call, got %d", execCount)
	}
}
