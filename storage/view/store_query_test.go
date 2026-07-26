package view

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

func TestSQLViewStore_Query_WhereOrderBy(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	views := []struct {
		key   string
		name  string
		email string
		age   int
	}{
		{"u1", "Alice", "alice@ex.com", 30},
		{"u2", "Bob", "bob@ex.com", 25},
		{"u3", "Charlie", "charlie@ex.com", 35},
		{"u4", "Diana", "diana@ex.com", 25},
	}

	for _, v := range views {
		if err := store.Set(
			ctx,
			testKey(v.key),
			&testView{Name: v.name, Email: v.email, Age: v.age},
		); err != nil {
			t.Fatalf("Set %s: %v", v.key, err)
		}
	}

	results, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "age", Op: kv.OpEq, Value: 25}},
		OrderBy:    "name",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Query age=25: got %d, want 2", len(results))
	}

	if results[0].Name != "Bob" || results[1].Name != "Diana" {
		t.Fatalf("Query order: got %s, %s; want Bob, Diana", results[0].Name, results[1].Name)
	}
}

func TestSQLViewStore_Query_Pagination(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("u%02d", i)
		if err := store.Set(ctx, testKey(key), &testView{
			Name: fmt.Sprintf("User%d", i), Email: "x@ex.com", Age: i,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	page1, err := store.Query(ctx, kv.ViewQuery{OrderBy: "key", Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("Query page 1: %v", err)
	}

	if len(page1) != 3 {
		t.Fatalf("Page 1: got %d, want 3", len(page1))
	}

	if page1[0].Age != 0 || page1[2].Age != 2 {
		t.Fatalf("Page 1 ages: got %d, %d; want 0, 2", page1[0].Age, page1[2].Age)
	}

	page2, err := store.Query(ctx, kv.ViewQuery{OrderBy: "key", Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("Query page 2: %v", err)
	}

	if page2[0].Age != 3 || page2[2].Age != 5 {
		t.Fatalf("Page 2 ages: got %d, %d; want 3, 5", page2[0].Age, page2[2].Age)
	}

	last, err := store.Query(ctx, kv.ViewQuery{OrderBy: "key", Limit: 3, Offset: 9})
	if err != nil {
		t.Fatalf("Query last page: %v", err)
	}

	if len(last) != 1 {
		t.Fatalf("Last page: got %d, want 1", len(last))
	}
}

func TestSQLViewStore_Query_Desc(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		if err := store.Set(
			ctx,
			testKey(name[:1]),
			&testView{Name: name, Email: "x@ex.com"},
		); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}

	results, err := store.Query(ctx, kv.ViewQuery{OrderBy: "name", Desc: true})
	if err != nil {
		t.Fatalf("Query desc: %v", err)
	}

	if results[0].Name != "Charlie" || results[2].Name != "Alice" {
		t.Fatalf("Desc order: got %s, ..., %s; want Charlie, ..., Alice",
			results[0].Name, results[2].Name)
	}
}

func TestSQLViewStore_QueryByTombstone(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	if err := store.Set(
		ctx,
		testKey("active1"),
		&testView{Name: "Alice", Email: "a@ex.com"},
	); err != nil {
		t.Fatalf("Set active1: %v", err)
	}

	if err := store.Set(
		ctx,
		testKey("active2"),
		&testView{Name: "Bob", Email: "b@ex.com"},
	); err != nil {
		t.Fatalf("Set active2: %v", err)
	}

	if err := store.Set(
		ctx,
		testKey("dead1"),
		&testView{Name: "Charlie", Email: "c@ex.com", Tombstoned: true},
	); err != nil {
		t.Fatalf("Set dead1: %v", err)
	}

	active, err := store.QueryByTombstone(ctx, true, false)
	if err != nil {
		t.Fatalf("QueryByTombstone exclude: %v", err)
	}

	if len(active) != 2 {
		t.Fatalf("Exclude tombstoned: got %d, want 2", len(active))
	}

	tombstoned, err := store.QueryByTombstone(ctx, false, true)
	if err != nil {
		t.Fatalf("QueryByTombstone only: %v", err)
	}

	if len(tombstoned) != 1 || tombstoned[0].Name != "Charlie" {
		t.Fatalf(
			"Only tombstoned: got %d, first=%s; want 1, Charlie",
			len(tombstoned),
			safeName(tombstoned),
		)
	}

	all, err := store.QueryByTombstone(ctx, false, false)
	if err != nil {
		t.Fatalf("QueryByTombstone all: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("All: got %d, want 3", len(all))
	}
}
