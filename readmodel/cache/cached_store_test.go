package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/cache/v2"
)

type testKey string

func (k testKey) String() string { return string(k) }

type testView struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func newTestStore(t *testing.T) (*readmodel.Store[testView, testKey], *kv.MemStore) {
	t.Helper()

	backend := kv.NewMemStore()
	t.Cleanup(func() { _ = backend.Close() })

	store := readmodel.New[testView, testKey](backend,
		readmodel.WithKeyPrefix[testView, testKey]("views:"),
	)

	return store, backend
}

func TestCachedStore_GetMissThenHit(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	// Set a value through the cached store
	if err := cached.Set(ctx, "1", &testView{Name: "alice", Age: 30}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// First Get: cache hit (value was cached on Set)
	got, err := cached.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get (cache hit): %v", err)
	}

	if got.Name != "alice" || got.Age != 30 {
		t.Fatalf("unexpected value: %+v", got)
	}

	// Second Get: still a cache hit
	got2, err := cached.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get (cache hit 2): %v", err)
	}

	if got2.Name != "alice" {
		t.Fatalf("unexpected value on second get: %+v", got2)
	}
}

func TestCachedStore_GetFromBackendOnMiss(t *testing.T) {
	t.Parallel()

	store, backend := newTestStore(t)
	ctx := context.Background()

	// Write directly to backend (bypassing cache)
	data, _ := codec.JSONCodec{}.Encode(&testView{Name: "bob", Age: 25})
	if err := backend.Set([]byte("views:2"), data); err != nil {
		t.Fatalf("backend.Set: %v", err)
	}

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	// Get: cache miss → delegates to store → caches result
	got, err := cached.Get(ctx, "2")
	if err != nil {
		t.Fatalf("Get (cache miss): %v", err)
	}

	if got.Name != "bob" || got.Age != 25 {
		t.Fatalf("unexpected value: %+v", got)
	}

	// Second Get: should be a cache hit now
	got2, err := cached.Get(ctx, "2")
	if err != nil {
		t.Fatalf("Get (cache hit): %v", err)
	}

	if got2.Name != "bob" {
		t.Fatalf("unexpected value on second get: %+v", got2)
	}
}

func TestCachedStore_NotFoundNotCached(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	// Get missing key: should return ErrNotFound
	_, err = cached.Get(ctx, "missing")
	if !errors.Is(err, readmodel.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Has should return false
	has, err := cached.Has(ctx, "missing")
	if err != nil {
		t.Fatalf("Has: %v", err)
	}

	if has {
		t.Fatal("expected Has to return false for missing key")
	}
}

func TestCachedStore_DeleteInvalidates(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	// Set and verify cached
	if err := cached.Set(ctx, "3", &testView{Name: "charlie"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := cached.Get(ctx, "3")
	if err != nil || got.Name != "charlie" {
		t.Fatalf("Get after Set: err=%v val=%+v", err, got)
	}

	// Delete should invalidate cache
	if err := cached.Delete(ctx, "3"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get after delete: should miss cache and return ErrNotFound from backend
	_, err = cached.Get(ctx, "3")
	if !errors.Is(err, readmodel.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestCachedStore_SetUpdatesCache(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	// Set initial value
	if err := cached.Set(ctx, "4", &testView{Name: "initial"}); err != nil {
		t.Fatalf("Set initial: %v", err)
	}

	// Update value
	if err := cached.Set(ctx, "4", &testView{Name: "updated"}); err != nil {
		t.Fatalf("Set updated: %v", err)
	}

	// Get should return updated value from cache
	got, err := cached.Get(ctx, "4")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "updated" {
		t.Fatalf("expected 'updated', got %q", got.Name)
	}
}

func TestCachedStore_TTLExpiration(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store,
		cache.WithTTL[testView, testKey](50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	if err := cached.Set(ctx, "5", &testView{Name: "ephemeral"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Should be cached immediately
	got, err := cached.Get(ctx, "5")
	if err != nil || got.Name != "ephemeral" {
		t.Fatalf("Get before TTL: err=%v val=%+v", err, got)
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should miss cache, re-fetch from backend (still exists in backend)
	got2, err := cached.Get(ctx, "5")
	if err != nil {
		t.Fatalf("Get after TTL: %v", err)
	}

	if got2.Name != "ephemeral" {
		t.Fatalf("expected 'ephemeral', got %q", got2.Name)
	}
}

func TestCachedStore_HasChecksCacheFirst(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	ctx := context.Background()

	cached, err := cache.New[testView, testKey](store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer cached.Close()

	if err := cached.Set(ctx, "6", &testView{Name: "exists"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Has should find it in cache
	has, err := cached.Has(ctx, "6")
	if err != nil || !has {
		t.Fatalf("Has (cached): err=%v has=%v", err, has)
	}
}

func TestCachedStore_NilStoreReturnsError(t *testing.T) {
	t.Parallel()

	_, err := cache.New[testView, testKey](nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}
