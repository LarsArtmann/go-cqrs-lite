package kv_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

type testID string

func (t testID) String() string { return string(t) }

type testUser struct {
	Name string
	Age  int
}

func TestTypedStore_GetSetDelete(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)

	ctx := context.Background()
	id := testID("user-1")

	// Initially not found.
	_, err := ts.Get(ctx, id)
	if err == nil {
		t.Fatal("expected error for missing key")
	}

	// Set and get back.
	err = ts.Set(ctx, id, &testUser{Name: "Alice", Age: 30})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := ts.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if val.Name != "Alice" || val.Age != 30 {
		t.Fatalf("got %+v, want {Alice 30}", val)
	}

	// Has.
	has, err := ts.Has(ctx, id)
	if err != nil {
		t.Fatalf("Has: %v", err)
	}

	if !has {
		t.Fatal("expected Has=true")
	}

	// Delete and verify gone.
	err = ts.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	has, err = ts.Has(ctx, id)
	if err != nil {
		t.Fatalf("Has after delete: %v", err)
	}

	if has {
		t.Fatal("expected Has=false after delete")
	}
}

func TestTypedStore_Scan(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store, kv.WithTypedKeyPrefix[testUser, testID]("users:"))

	ctx := context.Background()

	_ = ts.Set(ctx, testID("a"), &testUser{Name: "Alice"})
	_ = ts.Set(ctx, testID("b"), &testUser{Name: "Bob"})

	results, err := ts.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	names := []string{results[0].Name, results[1].Name}
	if names[0] != "Alice" || names[1] != "Bob" {
		t.Fatalf("got %v, want [Alice Bob]", names)
	}
}
