package storage_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func TestSQLViewStore_Count(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	for i := range 5 {
		key := testKey(string(rune('a' + i)))
		view := &testView{Name: string(rune('A' + i)), Age: i * 10}
		if i >= 3 {
			view.Tombstoned = true
		}
		if err := store.Set(ctx, key, view); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// Count all.
	count, err := store.Count(ctx, kv.ViewQuery{})
	if err != nil {
		t.Fatalf("Count all: %v", err)
	}
	if count != 5 {
		t.Fatalf("Count all: got %d, want 5", count)
	}

	// Count with filter.
	count, err = store.Count(ctx, kv.ViewQuery{
		Where: "age >= ?",
		Args:  []any{20},
	})
	if err != nil {
		t.Fatalf("Count filtered: %v", err)
	}
	if count != 3 {
		t.Fatalf("Count age>=20: got %d, want 3", count)
	}

	// Count tombstoned.
	count, err = store.Count(ctx, kv.ViewQuery{
		Where: "tombstoned != 0",
	})
	if err != nil {
		t.Fatalf("Count tombstoned: %v", err)
	}
	if count != 2 {
		t.Fatalf("Count tombstoned: got %d, want 2", count)
	}
}

func TestSQLViewStore_QueryFiltered(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	views := []struct {
		key    string
		name   string
		age    int
		active bool
	}{
		{"u1", "Alice", 30, true},
		{"u2", "Bob", 25, false},
		{"u3", "Charlie", 35, true},
		{"u4", "Diana", 25, true},
	}

	for _, v := range views {
		view := &testView{Name: v.name, Age: v.age, Tombstoned: !v.active}
		if err := store.Set(ctx, testKey(v.key), view); err != nil {
			t.Fatalf("Set %s: %v", v.key, err)
		}
	}

	// Structured filter: age = 25 AND active.
	results, err := store.QueryFiltered(
		ctx,
		kv.ViewFilter{
			Conditions: []kv.Condition{
				{Column: "age", Op: kv.OpEq, Value: 25},
				{Column: "tombstoned", Op: kv.OpEq, Value: false},
			},
		},
		kv.ViewQuery{OrderBy: "name"},
	)
	if err != nil {
		t.Fatalf("QueryFiltered: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Diana" {
		t.Fatalf("QueryFiltered: got %d results, first=%s; want 1, Diana",
			len(results), safeName(results))
	}

	// IN operator.
	results, err = store.QueryFiltered(
		ctx,
		kv.ViewFilter{
			Conditions: []kv.Condition{
				{Column: "name", Op: kv.OpIn, Value: []string{"Alice", "Charlie"}},
			},
		},
		kv.ViewQuery{OrderBy: "name"},
	)
	if err != nil {
		t.Fatalf("QueryFiltered IN: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("QueryFiltered IN: got %d, want 2", len(results))
	}

	// Range filter.
	results, err = store.QueryFiltered(
		ctx,
		kv.ViewFilter{
			Conditions: []kv.Condition{
				{Column: "age", Op: kv.OpGte, Value: 25},
				{Column: "age", Op: kv.OpLte, Value: 30},
			},
		},
		kv.ViewQuery{OrderBy: "age"},
	)
	if err != nil {
		t.Fatalf("QueryFiltered range: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("QueryFiltered range 25-30: got %d, want 3", len(results))
	}
}

func TestSQLViewStore_DeleteAll(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	for _, name := range []string{"a", "b", "c"} {
		if err := store.Set(ctx, testKey(name), &testView{Name: name}); err != nil {
			t.Fatalf("Set %s: %v", name, err)
		}
	}

	// Verify 3 records.
	all, err := store.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("Before DeleteAll: got %d, want 3", len(all))
	}

	// DeleteAll.
	if err := store.DeleteAll(ctx); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}

	// Verify 0 records.
	all, err = store.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan after DeleteAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("After DeleteAll: got %d, want 0", len(all))
	}

	// DeleteAll on empty table is a no-op.
	if err := store.DeleteAll(ctx); err != nil {
		t.Fatalf("DeleteAll empty: %v", err)
	}
}

func TestSQLViewStore_BatchSet(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	items := make([]kv.ViewItem[testView, testKey], 0, 100)
	for i := range 100 {
		items = append(items, kv.ViewItem[testView, testKey]{
			Key:   testKey(string(rune('a' + i))),
			Value: &testView{Name: string(rune('A' + i)), Age: i},
		})
	}

	if err := store.BatchSet(ctx, items); err != nil {
		t.Fatalf("BatchSet: %v", err)
	}

	count, err := store.Count(ctx, kv.ViewQuery{})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 100 {
		t.Fatalf("Count after BatchSet: got %d, want 100", count)
	}

	// Verify a sample record.
	got, err := store.Get(ctx, testKey("a"))
	if err != nil {
		t.Fatalf("Get a: %v", err)
	}
	if got.Name != "A" || got.Age != 0 {
		t.Fatalf("Get a: got %+v, want {A 0}", got)
	}

	// BatchSet with empty items is a no-op.
	if err := store.BatchSet(ctx, nil); err != nil {
		t.Fatalf("BatchSet empty: %v", err)
	}

	// BatchSet upserts existing records.
	items[0].Value.Age = 999
	if err := store.BatchSet(ctx, items[:1]); err != nil {
		t.Fatalf("BatchSet upsert: %v", err)
	}
	got, err = store.Get(ctx, testKey("a"))
	if err != nil {
		t.Fatalf("Get a after upsert: %v", err)
	}
	if got.Age != 999 {
		t.Fatalf("Get a after upsert: age=%d, want 999", got.Age)
	}
}

func TestSQLViewStore_Indexes(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mapper := testMapper()
	mapper.Indexes = []storage.IndexSpec{
		{Name: "idx_email", Columns: []string{"email"}},
		{Name: "idx_age", Columns: []string{"age"}},
	}

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, mapper)
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	for i := range 10 {
		if err := store.Set(ctx, testKey(string(rune('a'+i))),
			&testView{Name: string(rune('A' + i)), Email: "x@ex.com", Age: i}); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}

	// Query using indexed column should work.
	results, err := store.Query(ctx, kv.ViewQuery{
		Where:   "age >= ?",
		Args:    []any{5},
		OrderBy: "age",
	})
	if err != nil {
		t.Fatalf("Query indexed: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("Query age>=5: got %d, want 5", len(results))
	}

	// Verify indexes exist by querying sqlite_master.
	var indexCount int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name IN ('idx_email', 'idx_age')",
	).Scan(&indexCount)
	if err != nil {
		t.Fatalf("Query indexes: %v", err)
	}
	if indexCount != 2 {
		t.Fatalf("Index count: got %d, want 2", indexCount)
	}
}
