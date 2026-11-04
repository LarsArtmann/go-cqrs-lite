package kv_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

type stringKey string

func (s stringKey) String() string { return string(s) }

// fakeViewStore is a ViewStore that is NOT a *kv.TypedStore — the shape
// SQL-backed view stores (real columns) have.
type fakeViewStore struct {
	values map[string]*testView
	gets   int
}

type testView struct {
	Name  string
	Count int64
}

func newFakeViewStore() *fakeViewStore {
	return &fakeViewStore{values: map[string]*testView{}}
}

func (f *fakeViewStore) Get(_ context.Context, key stringKey) (*testView, error) {
	f.gets++

	val, ok := f.values[key.String()]
	if !ok {
		return nil, kv.ErrNotFound
	}

	copyVal := *val

	return &copyVal, nil
}

func (f *fakeViewStore) Set(_ context.Context, key stringKey, val *testView) error {
	copyVal := *val
	f.values[key.String()] = &copyVal

	return nil
}

func (f *fakeViewStore) Delete(_ context.Context, key stringKey) error {
	delete(f.values, key.String())

	return nil
}

func (f *fakeViewStore) Scan(_ context.Context, prefix []byte) ([]*testView, error) {
	out := make([]*testView, 0, len(f.values))

	for _, v := range f.values {
		out = append(out, v)
	}

	return out, nil
}

func TestNewCacheForViewStore_HitAndMiss(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newFakeViewStore()
	_ = store.Set(ctx, "k1", &testView{Name: "alpha", Count: 1})

	cache, err := kv.NewCacheForViewStore[testView, stringKey](store)
	if err != nil {
		t.Fatalf("NewCacheForViewStore: %v", err)
	}
	defer cache.Close()

	first, err := cache.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get miss: %v", err)
	}

	if first.Name != "alpha" {
		t.Fatalf("Get = %+v", first)
	}

	if _, err = cache.Get(ctx, "k1"); err != nil {
		t.Fatalf("Get hit: %v", err)
	}

	if store.gets != 1 {
		t.Errorf("second Get must be a cache hit, store saw %d gets", store.gets)
	}

	// Shallow-copy isolation: mutating the returned top-level value must not
	// corrupt the cached entry.
	first.Name = "mutated"

	again, err := cache.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get after mutation: %v", err)
	}

	if again.Name != "alpha" {
		t.Errorf("cached entry corrupted by caller mutation: %+v", again)
	}
}

func TestNewCacheForViewStore_HasFallsBackToGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newFakeViewStore()
	_ = store.Set(ctx, "k1", &testView{Name: "alpha"})

	cache, err := kv.NewCacheForViewStore[testView, stringKey](store)
	if err != nil {
		t.Fatalf("NewCacheForViewStore: %v", err)
	}
	defer cache.Close()

	has, err := cache.Has(ctx, "k1")
	if err != nil || !has {
		t.Fatalf("Has(k1) = %v, %v; want true", has, err)
	}

	has, err = cache.Has(ctx, "missing")
	if err != nil || has {
		t.Fatalf("Has(missing) = %v, %v; want false", has, err)
	}
}

func TestNewCacheForViewStore_WriteThroughAndDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newFakeViewStore()

	cache, err := kv.NewCacheForViewStore[testView, stringKey](store)
	if err != nil {
		t.Fatalf("NewCacheForViewStore: %v", err)
	}
	defer cache.Close()

	if err = cache.Set(ctx, "k1", &testView{Name: "beta"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	direct, err := store.Get(ctx, "k1")
	if err != nil || direct.Name != "beta" {
		t.Fatalf("write-through failed: %+v, %v", direct, err)
	}

	if err = cache.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err = cache.Get(ctx, "k1"); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
	}
}

func TestNewCacheForViewStore_NilStoreRejected(t *testing.T) {
	t.Parallel()

	if _, err := kv.NewCacheForViewStore[testView, stringKey](nil); err == nil {
		t.Fatal("nil ViewStore must be rejected")
	}
}

func TestNewCacheForViewStore_BackendAndStoreNil(t *testing.T) {
	t.Parallel()

	store := newFakeViewStore()

	cache, err := kv.NewCacheForViewStore[testView, stringKey](store)
	if err != nil {
		t.Fatalf("NewCacheForViewStore: %v", err)
	}
	defer cache.Close()

	if cache.Backend() != nil {
		t.Error("Backend() must be nil for non-TypedStore caches")
	}

	if cache.Store() != nil {
		t.Error("Store() must be nil for non-TypedStore caches")
	}
}

func TestWithCacheCopyFunc_DeepCopyForReferenceViews(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newFakeViewStore()
	_ = store.Set(ctx, "k1", &testView{Name: "alpha"})

	cache, err := kv.NewCacheForViewStore[testView, stringKey](
		store,
		kv.WithCacheCopyFunc[testView, stringKey](func(v *testView) *testView {
			out := *v

			return &out
		}),
	)
	if err != nil {
		t.Fatalf("NewCacheForViewStore: %v", err)
	}
	defer cache.Close()

	first, err := cache.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	first.Count = 99

	again, err := cache.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get hit: %v", err)
	}

	if again.Count == 99 {
		t.Error("custom copy func must isolate caller mutations")
	}
}
