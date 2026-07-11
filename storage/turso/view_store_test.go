package turso_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

type tursoViewKey string

func (k tursoViewKey) String() string { return string(k) }

type tursoView struct {
	Name       string
	Email      string
	Age        int
	Tombstoned bool
}

func tursoMapper() storage.ViewMapper[tursoView] {
	return storage.ViewMapper[tursoView]{
		Table: "turso_view_test",
		Columns: []storage.ViewColumn[tursoView]{
			{Name: "name", Type: "TEXT", Extract: func(v *tursoView) any { return v.Name }},
			{Name: "email", Type: "TEXT", Extract: func(v *tursoView) any { return v.Email }},
			{Name: "age", Type: "INTEGER", Extract: func(v *tursoView) any { return v.Age }},
			{
				Name:    "tombstoned",
				Type:    "INTEGER",
				Extract: func(v *tursoView) any { return v.Tombstoned },
			},
		},
		ScanRow: func(scan func(dest ...any) error) (*tursoView, error) {
			var v tursoView
			if err := scan(&v.Name, &v.Email, &v.Age, &v.Tombstoned); err != nil {
				return nil, err
			}

			return &v, nil
		},
		TombstoneColumn: "tombstoned",
	}
}

func setupTursoViewStore(
	t *testing.T,
) (*storage.SQLViewStore[tursoView, tursoViewKey], context.Context) {
	t.Helper()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	turso.ConfigurePool(db)

	store, err := turso.NewViewStore[tursoView, tursoViewKey](db, tursoMapper())
	if err != nil {
		t.Fatalf("NewViewStore: %v", err)
	}

	return store, context.Background()
}

func TestTursoViewStore_CRUD(t *testing.T) {
	t.Parallel()

	store, ctx := setupTursoViewStore(t)

	// Set.
	view := &tursoView{Name: "Alice", Email: "alice@ex.com", Age: 30}
	if err := store.Set(ctx, tursoViewKey("u1"), view); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get.
	got, err := store.Get(ctx, tursoViewKey("u1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "Alice" || got.Age != 30 {
		t.Fatalf("Get: got %+v, want Alice/30", got)
	}

	// Overwrite.
	view.Age = 31
	if err := store.Set(ctx, tursoViewKey("u1"), view); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}

	got, _ = store.Get(ctx, tursoViewKey("u1"))
	if got.Age != 31 {
		t.Fatalf("Overwrite: got age %d, want 31", got.Age)
	}

	// Delete.
	if err := store.Delete(ctx, tursoViewKey("u1")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Get(ctx, tursoViewKey("u1"))
	if err == nil {
		t.Fatal("Get after delete: expected error, got nil")
	}
}

func TestTursoViewStore_QueryConditions(t *testing.T) {
	t.Parallel()

	store, ctx := setupTursoViewStore(t)

	views := []struct {
		key  string
		name string
		age  int
	}{
		{"u1", "Alice", 30},
		{"u2", "Bob", 25},
		{"u3", "Charlie", 35},
		{"u4", "Diana", 25},
	}

	for _, v := range views {
		view := &tursoView{Name: v.name, Email: v.name + "@ex.com", Age: v.age}
		if err := store.Set(ctx, tursoViewKey(v.key), view); err != nil {
			t.Fatalf("Set %s: %v", v.key, err)
		}
	}

	// Query with structured conditions.
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

	// IN operator.
	results, err = store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{
			{Column: "name", Op: kv.OpIn, Values: []any{"Alice", "Charlie"}},
		},
		OrderBy: "name",
	})
	if err != nil {
		t.Fatalf("Query IN: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Query IN: got %d, want 2", len(results))
	}

	// Range filter.
	results, err = store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{
			{Column: "age", Op: kv.OpGte, Value: 25},
			{Column: "age", Op: kv.OpLte, Value: 30},
		},
		OrderBy: "age",
	})
	if err != nil {
		t.Fatalf("Query range: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Query range 25-30: got %d, want 3", len(results))
	}
}

func TestTursoViewStore_CountAndReset(t *testing.T) {
	t.Parallel()

	store, ctx := setupTursoViewStore(t)

	for i := range 5 {
		view := &tursoView{Name: "User", Age: i * 10}
		if err := store.Set(ctx, tursoViewKey(string(rune('a'+i))), view); err != nil {
			t.Fatalf("Set: %v", err)
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
		Conditions: []kv.Condition{{Column: "age", Op: kv.OpGte, Value: 20}},
	})
	if err != nil {
		t.Fatalf("Count filtered: %v", err)
	}

	if count != 3 {
		t.Fatalf("Count age>=20: got %d, want 3", count)
	}

	// DeleteAll.
	if err := store.DeleteAll(ctx); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}

	count, _ = store.Count(ctx, kv.ViewQuery{})
	if count != 0 {
		t.Fatalf("After DeleteAll: got %d, want 0", count)
	}
}

func TestTursoViewStore_QueryByTombstone(t *testing.T) {
	t.Parallel()

	store, ctx := setupTursoViewStore(t)

	if err := store.Set(ctx, tursoViewKey("alive"), &tursoView{Name: "Alive", Age: 1}); err != nil {
		t.Fatalf("Set alive: %v", err)
	}

	if err := store.Set(
		ctx,
		tursoViewKey("dead"),
		&tursoView{Name: "Dead", Age: 2, Tombstoned: true},
	); err != nil {
		t.Fatalf("Set dead: %v", err)
	}

	active, err := store.QueryByTombstone(ctx, true, false)
	if err != nil {
		t.Fatalf("QueryByTombstone exclude: %v", err)
	}

	if len(active) != 1 || active[0].Name != "Alive" {
		t.Fatalf("Exclude tombstoned: got %d results, want 1 (Alive)", len(active))
	}

	dead, err := store.QueryByTombstone(ctx, false, true)
	if err != nil {
		t.Fatalf("QueryByTombstone only: %v", err)
	}

	if len(dead) != 1 || dead[0].Name != "Dead" {
		t.Fatalf("Only tombstoned: got %d results, want 1 (Dead)", len(dead))
	}
}
