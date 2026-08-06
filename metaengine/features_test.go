package metaengine

import (
	"context"
	"database/sql"
	"strings"
	"testing"

)

type (
	testTaskID string
	testTask   struct {
		ID     testTaskID
		Title  string
		Status string
	}
)

type testFindTask struct {
	ID testTaskID
}

func testTaskQuery() QueryDecl[testFindTask, testTask] {
	return Query[testFindTask, testTask](
		"tasks",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
	)
}

func newMemoryTestStore(t *testing.T) *Store {
	t.Helper()

	eng := NewMemoryEngine()

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func newSQLiteTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	eng, err := newMemoryEngineForTest()
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

func TestApplyBatch(t *testing.T) {
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	events := []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "Task 1", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "Task 2", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t3", Title: "Task 3", Status: "done"}},
	}

	if err := store.ApplyBatch(t.Context(), events); err != nil {
		t.Fatalf("ApplyBatch failed: %v", err)
	}

	reader := NewReader[testTask](store, "tasks")
	results, err := reader.Scan(t.Context())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestCollections(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].Name != "tasks" {
		t.Errorf("expected collection name 'tasks', got %q", collections[0].Name)
	}

	if collections[0].ADT != ADTMap {
		t.Errorf("expected ADT Map, got %s", collections[0].ADT)
	}
}


func _skipped_sqlite_0(t *testing.T) {
	t.Skip("requires SQLite engine — moved to sqliteengine module after ADR-0115")
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()

	store, err := Plan(
		[]Engine{eng},
		WithDryRun(),
		Query[testFindTask, testTask](
			"find_filtered",
			OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
				return e.ID, e
			}),
			FilterOnField[testTask]("status", FilterEq),
			SortOnField[testTask]("title", false),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	if store.Plan() == nil {
		t.Fatal("expected non-nil plan in dry-run mode")
	}

	if len(store.Plan().LayoutPlans) == 0 {
		t.Error("expected at least one LayoutPlan in dry-run mode")
	}

	var tableName string
	_ = db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'meta_planned_%' LIMIT 1",
	).Scan(&tableName)
	if tableName != "" {
		t.Errorf("expected no planned tables in dry-run mode, found %q", tableName)
	}
}

func TestExplain(t *testing.T) {
	t.Skip("requires SQLite EXPLAIN — see sqliteengine module")
	t.Parallel()

	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng := NewMemoryEngine()

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	reader := NewReader[testTask](store, "tasks")

	query, args := reader.Explain(
		context.Background(),
		WithFilter("title", FilterEq, "test"),
		WithLimit(10),
	)

	if query == "" {
		t.Error("expected non-empty EXPLAIN query")
	}

	if len(args) == 0 {
		t.Error("expected non-empty args")
	}

	if !strings.Contains(query, "SELECT") {
		t.Errorf("expected SELECT in query, got %q", query)
	}
}

func TestIsPoisoned(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	if err := store.IsPoisoned("tasks"); err != nil {
		t.Errorf("expected nil for healthy collection, got %v", err)
	}
}

func TestExportedErrors(t *testing.T) {
	t.Parallel()

	if ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}

	if ErrAmbiguousKey == nil {
		t.Error("ErrAmbiguousKey should not be nil")
	}

	if ErrUnsupportedADT == nil {
		t.Error("ErrUnsupportedADT should not be nil")
	}

	if ErrLayoutConflict == nil {
		t.Error("ErrLayoutConflict should not be nil")
	}
}
