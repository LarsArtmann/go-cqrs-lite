package kv_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

func TestCache_GetSetDelete(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	id := testID("u1")

	err = cache.Set(ctx, id, &testUser{Name: "Alice", Age: 30})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if val.Name != "Alice" {
		t.Fatalf("got %s, want Alice", val.Name)
	}

	err = cache.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = cache.Get(ctx, id)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCache_NilStore(t *testing.T) {
	t.Parallel()

	_, err := kv.NewCache[testUser, testID](nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestCache_InvalidCapacity(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	_, err := kv.NewCache[testUser, testID](ts, kv.WithCacheCapacity[testUser, testID](0))
	if err == nil {
		t.Fatal("expected error for zero capacity")
	}
}

func TestCache_Has(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	id := testID("user-1")
	ctx := context.Background()

	exists, err := cache.Has(ctx, id)
	if err != nil {
		t.Fatalf("Has before Set: %v", err)
	}
	if exists {
		t.Fatal("should not exist before Set")
	}

	_ = cache.Set(ctx, id, &testUser{Name: "Alice"})

	exists, err = cache.Has(ctx, id)
	if err != nil {
		t.Fatalf("Has after Set: %v", err)
	}
	if !exists {
		t.Fatal("should exist after Set")
	}
}

func TestCache_Scan(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	_ = cache.Set(ctx, testID("user-1"), &testUser{Name: "Alice"})
	_ = cache.Set(ctx, testID("user-2"), &testUser{Name: "Bob"})

	results, err := cache.Scan(ctx, []byte("user-"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestCache_BackendAndStore(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	if cache.Backend() == nil {
		t.Fatal("Backend should not be nil")
	}
	if cache.Store() == nil {
		t.Fatal("Store should not be nil")
	}
}

func TestCache_DeleteInvalidates(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	id := testID("user-1")
	_ = cache.Set(ctx, id, &testUser{Name: "Alice"})
	_ = cache.Delete(ctx, id)

	_, err = cache.Get(ctx, id)
	if err == nil {
		t.Fatal("Get after Delete should return error")
	}
}

func TestCache_GetReturnsIsolatedCopy(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	id := testID("user-1")

	first, err := cache.Get(ctx, id)
	if err == nil {
		t.Fatal("expected miss before Set")
	}

	_ = cache.Set(ctx, id, &testUser{Name: "Alice", Age: 30})

	// Miss path: the returned value is fresh from the store.
	first, err = cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (miss path): %v", err)
	}

	first.Name = "Mutated"
	first.Age = 99

	second, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (hit path): %v", err)
	}

	if second.Name != "Alice" || second.Age != 30 {
		t.Fatalf("mutation leaked through cache hit: got %+v, want {Alice 30}", *second)
	}

	// Hit path: the returned value is a copy of the cached entry.
	second.Name = "MutatedAgain"

	third, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (hit path 2): %v", err)
	}

	if third.Name != "Alice" {
		t.Fatalf("mutation of a hit-path result leaked: got %s, want Alice", third.Name)
	}
}

func TestCache_SetDoesNotShareValueWithCache(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	id := testID("user-1")

	val := &testUser{Name: "Alice", Age: 30}
	if err := cache.Set(ctx, id, val); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val.Name = "MutatedAfterSet"
	val.Age = 99

	got, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}

	if got.Name != "Alice" || got.Age != 30 {
		t.Fatalf("post-Set mutation leaked into cache: got %+v, want {Alice 30}", *got)
	}
}

func TestCache_TwoReadersGetDistinctValues(t *testing.T) {
	t.Parallel()

	store := kv.NewMemStore()
	defer store.Close()

	ts := kv.NewTypedStore[testUser, testID](store)
	cache, err := kv.NewCache[testUser, testID](ts)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	id := testID("user-1")

	if err := cache.Set(ctx, id, &testUser{Name: "Alice"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	a, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get a: %v", err)
	}
	b, err := cache.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get b: %v", err)
	}

	if a == b {
		t.Fatal("two concurrent readers received the same *T — shared pointer hazard")
	}
}
