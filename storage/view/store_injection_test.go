package view

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// hostileColumns are column references that attempt to smuggle SQL syntax.
// Every one of them must be rejected by the store's allowlist, never reach
// the SQL layer.
var hostileColumns = []string{
	"name; DROP TABLE test_views",
	"name)--",
	"1=1) OR (1",
	"name UNION SELECT email FROM test_views",
	"age--",
	"*",
	"name ASC, email",
	"n'a'me",
}

func TestSQLViewStore_Query_RejectsHostileConditionColumns(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		_, err := store.Query(ctx, kv.ViewQuery{
			Conditions: []kv.Condition{{Column: column, Op: kv.OpEq, Value: "x"}},
		})
		if err == nil {
			t.Errorf("Query with hostile condition column %q: want rejection, got nil", column)
		}
	}
}

func TestSQLViewStore_Query_RejectsHostileOrderColumns(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		_, err := store.Query(ctx, kv.ViewQuery{OrderBy: column})
		if err == nil {
			t.Errorf("Query with hostile order column %q: want rejection, got nil", column)
		}
	}
}

func TestSQLViewStore_Query_RejectsHostileMultiOrderColumns(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		_, err := store.Query(ctx, kv.ViewQuery{
			Order: []kv.OrderClause{{Column: "name"}, {Column: column}},
		})
		if err == nil {
			t.Errorf("Query with hostile multi-order column %q: want rejection, got nil", column)
		}
	}
}

func TestSQLViewStore_Query_RejectsHostileKeysetColumns(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		_, err := store.Query(ctx, kv.ViewQuery{
			Keyset: &kv.Keyset{Columns: []string{column}, Values: []any{"x"}},
		})
		if err == nil {
			t.Errorf("Query with hostile keyset column %q: want rejection, got nil", column)
		}
	}
}

func TestSQLViewStore_Query_RejectsUnsupportedOperator(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	_, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{
			Column: "name",
			Op:     "= 'x' OR 1=1 --",
			Value:  "x",
		}},
	})
	if err == nil {
		t.Fatal("Query with hostile operator: want rejection, got nil")
	}
}

func TestSQLViewStore_Count_RejectsHostileInputs(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		_, err := store.Count(ctx, kv.ViewQuery{
			Conditions: []kv.Condition{{Column: column, Op: kv.OpEq, Value: "x"}},
		})
		if err == nil {
			t.Errorf("Count with hostile column %q: want rejection, got nil", column)
		}
	}

	_, err := store.Count(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{
			Column: "name",
			Op:     "IS NOT NULL; DROP TABLE test_views",
		}},
	})
	if err == nil {
		t.Fatal("Count with hostile operator: want rejection, got nil")
	}
}

func TestSQLViewStore_RejectionNamesOffendingColumn(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	_, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "secret_column", Op: kv.OpEq, Value: 1}},
	})
	if err == nil {
		t.Fatal("Query with undeclared column: want rejection")
	}

	if !strings.Contains(err.Error(), "secret_column") {
		t.Errorf("rejection should name the offending column, got: %v", err)
	}
}

func TestSQLViewStore_LegitQueriesStillPass(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	if err := store.Set(
		ctx,
		testKey("u1"),
		&testView{Name: "A", Email: "a@ex.com", Age: 1},
	); err != nil {
		t.Fatalf("Set: %v", err)
	}

	results, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "age", Op: kv.OpGte, Value: 0}},
		OrderBy:    "name",
	})
	if err != nil {
		t.Fatalf("Query with legitimate condition: %v", err)
	}

	if len(results) != 1 || results[0].Name != "A" {
		t.Fatalf("Query results: got %+v, want single A", results)
	}

	count, err := store.Count(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "tombstoned", Op: kv.OpEq, Value: false}},
	})
	if err != nil {
		t.Fatalf("Count on tombstone column: %v", err)
	}

	if count != 1 {
		t.Fatalf("Count: got %d, want 1", count)
	}
}
