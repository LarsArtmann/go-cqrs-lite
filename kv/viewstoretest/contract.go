// Package viewstoretest provides a contract test suite for [kv.ViewStore]
// implementations. Any type that implements [kv.ViewStore] can verify
// correctness by calling [RunSuite].
//
// Usage:
//
//	func TestTypedStoreContract(t *testing.T) {
//	    viewstoretest.RunSuite(testView{}, t, viewstoretest.Config[testView, testKey]{
//	        Factory:   func(t *testing.T) kv.ViewStore[testView, testKey] { ... },
//	        MakeKey:   func(s string) testKey { return testKey(s) },
//	        MakeValue: func(name string) *testView { return &testView{Name: name} },
//	        Equal:     func(a, b *testView) bool { return a.Name == b.Name },
//	    })
//	}
package viewstoretest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

const (
	scanRecordCount  = 3
	countRecordCount = 5
)

// Config parameterises the contract test suite with type-specific factories.
type Config[V any, K fmt.Stringer] struct {
	// Factory returns a fresh, empty store. Called once per subtest.
	Factory func(t *testing.T) kv.ViewStore[V, K]

	// MakeKey converts a string to the key type K.
	MakeKey func(s string) K

	// MakeValue creates a *V with the given name for testing.
	MakeValue func(name string) *V

	// Equal reports whether two *V values are equivalent.
	Equal func(a, b *V) bool
}

// RunSuite runs the full [kv.ViewStore] contract against the implementation
// returned by cfg.Factory. Each subtest is independent and calls Factory
// to get a fresh store.
func RunSuite[V any, K fmt.Stringer](t *testing.T, cfg Config[V, K]) {
	t.Helper()

	ctx := context.Background()

	t.Run("GetMissing", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)

		_, err := store.Get(ctx, cfg.MakeKey("nonexistent"))
		if !errors.Is(err, kv.ErrNotFound) {
			t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("SetGet", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)
		val := cfg.MakeValue("Alice")

		err := store.Set(ctx, cfg.MakeKey("u1"), val)
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		got, err := store.Get(ctx, cfg.MakeKey("u1"))
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if !cfg.Equal(got, val) {
			t.Fatalf("Get: got %+v, want %+v", got, val)
		}
	})

	t.Run("SetOverwrite", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)
		key := cfg.MakeKey("u1")

		err := store.Set(ctx, key, cfg.MakeValue("Alice"))
		if err != nil {
			t.Fatalf("Set initial: %v", err)
		}

		updated := cfg.MakeValue("Bob")

		err = store.Set(ctx, key, updated)
		if err != nil {
			t.Fatalf("Set overwrite: %v", err)
		}

		got, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if !cfg.Equal(got, updated) {
			t.Fatalf("Get after overwrite: got %+v, want %+v", got, updated)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)
		key := cfg.MakeKey("u1")

		err := store.Set(ctx, key, cfg.MakeValue("Alice"))
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		err = store.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = store.Get(ctx, key)
		if !errors.Is(err, kv.ErrNotFound) {
			t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteMissing", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)

		err := store.Delete(ctx, cfg.MakeKey("never-existed"))
		if err != nil {
			t.Fatalf("Delete missing: err = %v, want nil", err)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)

		for _, name := range []string{"Charlie", "Alice", "Bob"} {
			err := store.Set(ctx, cfg.MakeKey(name), cfg.MakeValue(name))
			if err != nil {
				t.Fatalf("Set %s: %v", name, err)
			}
		}

		all, err := store.Scan(ctx, nil)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}

		if len(all) != scanRecordCount {
			t.Fatalf("Scan: got %d records, want %d", len(all), scanRecordCount)
		}
	})

	t.Run("ScanEmpty", func(t *testing.T) {
		t.Parallel()

		store := cfg.Factory(t)

		all, err := store.Scan(ctx, nil)
		if err != nil {
			t.Fatalf("Scan empty: %v", err)
		}

		if len(all) != 0 {
			t.Fatalf("Scan empty: got %d, want 0", len(all))
		}
	})
}

// RunOptionalSuite tests optional capabilities (ViewQuerier, ViewCounter,
// ViewResetter, ViewBatchSetter) if the store implements them.
// Stores that only implement [kv.ViewStore] will skip these tests.
func RunOptionalSuite[V any, K fmt.Stringer](t *testing.T, cfg Config[V, K]) {
	t.Helper()

	ctx := context.Background()

	t.Run("Resetter", func(t *testing.T) {
		store := cfg.Factory(t)

		resetter, ok := store.(kv.ViewResetter[V])
		if !ok {
			t.Skip("store does not implement ViewResetter")
		}

		for i := range scanRecordCount {
			err := store.Set(ctx, cfg.MakeKey(fmt.Sprintf("k%d", i)), cfg.MakeValue("val"))
			if err != nil {
				t.Fatalf("Set: %v", err)
			}
		}

		err := resetter.DeleteAll(ctx)
		if err != nil {
			t.Fatalf("DeleteAll: %v", err)
		}

		all, err := store.Scan(ctx, nil)
		if err != nil {
			t.Fatalf("Scan after DeleteAll: %v", err)
		}

		if len(all) != 0 {
			t.Fatalf("After DeleteAll: got %d, want 0", len(all))
		}
	})

	t.Run("Counter", func(t *testing.T) {
		store := cfg.Factory(t)

		counter, ok := store.(kv.ViewCounter[V])
		if !ok {
			t.Skip("store does not implement ViewCounter")
		}

		for i := range countRecordCount {
			err := store.Set(ctx, cfg.MakeKey(fmt.Sprintf("k%d", i)), cfg.MakeValue("val"))
			if err != nil {
				t.Fatalf("Set: %v", err)
			}
		}

		count, err := counter.Count(ctx, kv.ViewQuery{}) //nolint:exhaustruct // count all
		if err != nil {
			t.Fatalf("Count: %v", err)
		}

		if count != countRecordCount {
			t.Fatalf("Count: got %d, want %d", count, countRecordCount)
		}
	})
}
