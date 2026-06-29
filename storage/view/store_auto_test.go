package view

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

type autoView struct {
	Name       string `view:"name"`
	Email      string `view:"email"`
	Age        int    `view:"age"`
	Tombstoned bool   `view:"tombstoned"`
	SkipMe     string // no tag → not a column
}

func (v *autoView) IsTombstoned() bool { return v.Tombstoned }

func TestSQLViewStore_AutoMapper(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mapper := AutoMapperWithTombstone[autoView]("auto_views", "tombstoned")
	store, err := NewSQLiteViewStore[autoView, testKey](db, mapper)
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	// Set + Get roundtrip.
	view := &autoView{Name: "Alice", Email: "alice@ex.com", Age: 30, SkipMe: "hidden"}
	if err := store.Set(ctx, testKey("u1"), view); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, testKey("u1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@ex.com" || got.Age != 30 {
		t.Fatalf("Get: got %+v, want Alice", got)
	}

	// SkipMe should not be a column — it won't be stored.
	if got.SkipMe != "" {
		t.Fatalf("SkipMe should be zero, got %q", got.SkipMe)
	}

	// Query by auto-mapped column.
	results, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "age", Op: kv.OpEq, Value: 30}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Alice" {
		t.Fatalf("Query age=30: got %d results", len(results))
	}

	// Tombstone filtering.
	view.Tombstoned = true
	if err := store.Set(ctx, testKey("u1"), view); err != nil {
		t.Fatalf("Set tombstoned: %v", err)
	}

	tombstoned, err := store.QueryByTombstone(ctx, false, true)
	if err != nil {
		t.Fatalf("QueryByTombstone: %v", err)
	}
	if len(tombstoned) != 1 {
		t.Fatalf("Tombstoned: got %d, want 1", len(tombstoned))
	}
}
