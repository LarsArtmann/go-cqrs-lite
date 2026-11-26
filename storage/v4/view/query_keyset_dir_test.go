package view

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// The cursor's direction is that of its own ORDER BY clause, not of the
// clause at the same index in the sort.
func TestBuildKeysetClause_DirectionFollowsOwnColumn(t *testing.T) {
	t.Parallel()

	order := []kv.OrderClause{{Column: "age", Desc: false}, {Column: "key", Desc: true}}

	sql, args := buildKeysetClause([]string{"key"}, []any{"u3"}, order,
		sqlpkg.SQLiteDialect{}.Placeholder, 1)

	if sql != "(key < ?)" {
		t.Errorf("clause = %s, want (key < ?)", sql)
	}

	if len(args) != 1 || args[0] != "u3" {
		t.Errorf("args = %v, want [u3]", args)
	}
}

func TestSQLViewStore_Query_KeysetSubsetColumnDirection(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	rows := map[string]int{"u1": 10, "u2": 20, "u3": 30, "u4": 40}
	for key, age := range rows {
		if err := store.Set(ctx, testKey(key), &testView{Name: key, Age: age}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	q := kv.ViewQuery{
		Order:  []kv.OrderClause{{Column: "age", Desc: false}, {Column: "key", Desc: true}},
		Keyset: &kv.Keyset{Columns: []string{"key"}, Values: []any{"u3"}},
	}

	got, err := store.Query(ctx, q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	// ORDER BY age ASC puts u1, u2 first; the cursor seeks past u3 in key
	// order, so only the keys below u3 remain.
	if len(got) != 2 || got[0].Name != "u1" || got[1].Name != "u2" {
		names := make([]string, 0, len(got))
		for _, v := range got {
			names = append(names, v.Name)
		}

		t.Fatalf("page after key u3: got %v, want [u1 u2]", names)
	}
}
