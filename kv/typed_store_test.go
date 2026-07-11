package kv_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
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

func TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ctx := context.Background()
	id := testID("legacy-1")

	// Simulate pre-envelope data: raw JSON bytes written directly to the backend
	// (exactly what the old JSONCodec default would have produced).
	rawJSON := []byte(`{"Name":"Legacy","Age":42}`)
	if err := store.Set(ctx, []byte(id), rawJSON); err != nil {
		t.Fatalf("seed raw JSON: %v", err)
	}

	// New TypedStore defaults to CBORCodec — but UnwrapDecode falls back to JSON.
	ts := kv.NewTypedStore[testUser, testID](store)

	val, err := ts.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get legacy data through new CBOR-default store: %v", err)
	}
	if val.Name != "Legacy" || val.Age != 42 {
		t.Fatalf("got %+v, want {Legacy 42}", val)
	}
}

func TestTypedStore_Migration_MixedOldAndNewData(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ctx := context.Background()

	// Old key: raw JSON (pre-envelope legacy data).
	oldID := testID("old")
	rawJSON := []byte(`{"Name":"OldFormat","Age":10}`)
	_ = store.Set(ctx, []byte(oldID), rawJSON)

	// New key: envelope-wrapped CBOR (post-flip data, written through TypedStore).
	newID := testID("new")
	ts := kv.NewTypedStore[testUser, testID](store)
	if err := ts.Set(ctx, newID, &testUser{Name: "NewFormat", Age: 20}); err != nil {
		t.Fatalf("Set new data: %v", err)
	}

	// Both must read correctly through the same store.
	oldVal, err := ts.Get(ctx, oldID)
	if err != nil {
		t.Fatalf("Get old raw-JSON key: %v", err)
	}
	if oldVal.Name != "OldFormat" || oldVal.Age != 10 {
		t.Fatalf("old data: got %+v, want {OldFormat 10}", oldVal)
	}

	newVal, err := ts.Get(ctx, newID)
	if err != nil {
		t.Fatalf("Get new envelope-CBOR key: %v", err)
	}
	if newVal.Name != "NewFormat" || newVal.Age != 20 {
		t.Fatalf("new data: got %+v, want {NewFormat 20}", newVal)
	}

	// Scan must also handle mixed formats.
	results, err := ts.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan mixed data: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestTypedStore_Scan(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](
		store,
		kv.WithTypedKeyPrefix[testUser, testID]("users:"),
	)

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
