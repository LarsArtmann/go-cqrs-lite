package metaengine_test

import (
	"context"
	"database/sql"
	"testing"


	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestTypedReader_Get exercises TypedReader[V].Get across memory and SQLite
// engines, verifying that typed point lookups work identically.
func TestTypedReader_Get(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	for _, tc := range []struct {
		name string
		eng  metaengine.Engine
	}{
		{"memory", metaengine.NewMemoryEngine()},
		{"sqlite", newIsolatedSQLiteEngine(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store, err := metaengine.Plan(
				[]metaengine.Engine{tc.eng},
				findTaskQuery(),
			)
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			defer store.Close()

			if err := store.Apply(ctx, "TaskCreated", TaskCreated{
				ID: "t1", Title: "Test", Status: "open", Priority: 5,
			}); err != nil {
				t.Fatalf("Apply: %v", err)
			}

			reader := metaengine.NewReader[FindTaskResult](store, "find_task")

			// Existing key.
			got, found, err := reader.Get(ctx, TaskID("t1"))
			if err != nil {
				t.Fatalf("Get: %v", err)
			}

			if !found {
				t.Fatal("Get: not found, want found")
			}

			if got.Title != "Test" {
				t.Fatalf("Get: Title = %q, want %q", got.Title, "Test")
			}

			if got.Priority != 5 {
				t.Fatalf("Get: Priority = %d, want 5", got.Priority)
			}

			// Missing key.
			_, found, err = reader.Get(ctx, TaskID("nonexistent"))
			if err != nil {
				t.Fatalf("Get(missing): %v", err)
			}

			if found {
				t.Fatal("Get(missing): found, want not found")
			}
		})
	}
}

// TestTypedReader_Scan exercises TypedReader[V].Scan with filter/sort/limit
// options across memory and SQLite engines.
func TestTypedReader_Scan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	// Use FilterOnField/SortOnField so the auto-layout kicks in.
	q := metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
		"typed_scan_tasks",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
			return e.ID, FindTaskResult{
				ID: e.ID, Title: e.Title, Status: e.Status, Priority: e.Priority,
			}
		}),
		metaengine.FilterOnField[FindTaskResult]("Status", metaengine.FilterEq),
		metaengine.SortOnField[FindTaskResult]("Priority", true),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	for _, tc := range []TaskCreated{
		{ID: "a", Title: "A", Status: "open", Priority: 1},
		{ID: "b", Title: "B", Status: "open", Priority: 3},
		{ID: "c", Title: "C", Status: "done", Priority: 2},
		{ID: "d", Title: "D", Status: "open", Priority: 5},
	} {
		if err := store.Apply(ctx, "TaskCreated", tc); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	reader := metaengine.NewReader[FindTaskResult](store, "typed_scan_tasks")

	// Scan with filter + sort + limit.
	results, err := reader.Scan(
		ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
		metaengine.WithSort("Priority", true),
		metaengine.WithLimit(10),
	)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Scan: got %d results, want 3 (open only)", len(results))
	}

	// Descending priority: d(5), b(3), a(1).
	wantIDs := []TaskID{"d", "b", "a"}
	for i, r := range results {
		if r.ID != wantIDs[i] {
			t.Fatalf("Scan[%d]: ID = %s, want %s", i, r.ID, wantIDs[i])
		}
	}

	// Scan with limit.
	limited, err := reader.Scan(
		ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
		metaengine.WithSort("Priority", true),
		metaengine.WithLimit(2),
	)
	if err != nil {
		t.Fatalf("Scan(limit=2): %v", err)
	}

	if len(limited) != 2 {
		t.Fatalf("Scan(limit=2): got %d, want 2", len(limited))
	}
}

// TestTypedReader_Exists exercises TypedReader[V].Exists for set membership.
func TestTypedReader_Exists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		checkAssigneeQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	if err := store.Apply(ctx, "TaskAssigned", TaskAssigned{
		TaskID: "t1", Assignee: "alice",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	reader := metaengine.NewReader[bool](store, "check_assignee")

	exists, err := reader.Exists(ctx, UserID("alice"))
	if err != nil {
		t.Fatalf("Exists(alice): %v", err)
	}

	if !exists {
		t.Fatal("Exists(alice): false, want true")
	}

	exists, err = reader.Exists(ctx, UserID("bob"))
	if err != nil {
		t.Fatalf("Exists(bob): %v", err)
	}

	if exists {
		t.Fatal("Exists(bob): true, want false")
	}
}
