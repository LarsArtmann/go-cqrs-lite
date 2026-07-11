package view

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

type testKey string

func (k testKey) String() string { return string(k) }

type testView struct {
	Name       string
	Email      string
	Age        int
	Tombstoned bool
}

func (v *testView) IsTombstoned() bool { return v.Tombstoned }

func testMapper() ViewMapper[testView] {
	return ViewMapper[testView]{
		Table: "test_views",
		Columns: []ViewColumn[testView]{
			{Name: "name", Type: "TEXT", Extract: func(v *testView) any { return v.Name }},
			{Name: "email", Type: "TEXT", Extract: func(v *testView) any { return v.Email }},
			{Name: "age", Type: "INTEGER", Extract: func(v *testView) any { return v.Age }},
			{
				Name:    "tombstoned",
				Type:    "INTEGER",
				Extract: func(v *testView) any { return v.Tombstoned },
			},
		},
		ScanRow: func(scan func(dest ...any) error) (*testView, error) {
			var v testView
			var tombInt int
			if err := scan(&v.Name, &v.Email, &v.Age, &tombInt); err != nil {
				return nil, err
			}
			v.Tombstoned = tombInt != 0
			return &v, nil
		},
		TombstoneColumn: "tombstoned",
	}
}

func newTestViewStore(t *testing.T) *SQLViewStore[testView, testKey] {
	t.Helper()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	return store
}

func safeName(views []*testView) string {
	if len(views) == 0 {
		return "(empty)"
	}

	return views[0].Name
}

func TestSQLViewStore_CRUD(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	// Get on missing key → ErrNotFound.
	if _, err := store.Get(ctx, testKey("missing")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}

	// Set + Get roundtrip.
	view := &testView{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := store.Set(ctx, testKey("u1"), view); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, testKey("u1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@example.com" || got.Age != 30 {
		t.Fatalf("Get: got %+v, want Alice", got)
	}

	// Set overwrite (upsert).
	view.Age = 31
	if err := store.Set(ctx, testKey("u1"), view); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}

	got, err = store.Get(ctx, testKey("u1"))
	if err != nil {
		t.Fatalf("Get after overwrite: %v", err)
	}

	if got.Age != 31 {
		t.Fatalf("Get after overwrite: age = %d, want 31", got.Age)
	}

	// Delete.
	if err := store.Delete(ctx, testKey("u1")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(ctx, testKey("u1")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}

	// Delete missing is a no-op.
	if err := store.Delete(ctx, testKey("never-existed")); err != nil {
		t.Fatalf("Delete missing: err = %v, want nil", err)
	}
}

func TestSQLViewStore_SetNil(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)

	if err := store.Set(context.Background(), testKey("nil"), nil); err == nil {
		t.Fatal("Set nil: expected error, got nil")
	}
}

func TestSQLViewStore_Scan(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	views := map[string]*testView{
		"c": {Name: "Charlie", Email: "c@ex.com", Age: 25},
		"a": {Name: "Alice", Email: "a@ex.com", Age: 30},
		"b": {Name: "Bob", Email: "b@ex.com", Age: 28},
	}

	for k, v := range views {
		if err := store.Set(ctx, testKey(k), v); err != nil {
			t.Fatalf("Set %s: %v", k, err)
		}
	}

	// Scan all → ordered by key.
	all, err := store.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("Scan: got %d records, want 3", len(all))
	}

	if all[0].Name != "Alice" || all[1].Name != "Bob" || all[2].Name != "Charlie" {
		t.Fatalf("Scan order: got %s, %s, %s; want Alice, Bob, Charlie",
			all[0].Name, all[1].Name, all[2].Name)
	}

	// Scan with prefix.
	prefix, err := store.Scan(ctx, []byte("a"))
	if err != nil {
		t.Fatalf("Scan prefix: %v", err)
	}

	if len(prefix) != 1 || prefix[0].Name != "Alice" {
		t.Fatalf(
			"Scan prefix 'a': got %d records, first=%s; want 1, Alice",
			len(prefix),
			safeName(prefix),
		)
	}
}

func TestSQLViewStore_ImplementsInterfaces(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)

	var _ kv.ViewStore[testView, testKey] = store
	var _ kv.ViewQuerier[testView] = store
	var _ kv.TombstoneQuerier[testView] = store
}
