package kv_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
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
