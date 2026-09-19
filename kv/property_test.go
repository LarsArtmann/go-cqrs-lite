package kv_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// Property tests for kv.TypedStore[T,K] and Cache[T,K] invariants.
// Mirrors the idempotency property-test pattern (rapid.Check + per-store factory).

// storeFactory builds a fresh TypedStore + cleanup for each rapid iteration.
type storeFactory func() (*kv.TypedStore[testUser, testID], func())

func allTypedStores() map[string]storeFactory {
	return map[string]storeFactory{
		"memstore": func() (*kv.TypedStore[testUser, testID], func()) {
			s := kv.NewMemStore()

			return kv.NewTypedStore[testUser, testID](s), func() { _ = s.Close() }
		},
	}
}

// genUser generates a random testUser via rapid.
func genUser() *rapid.Generator[testUser] {
	return rapid.Custom(func(t *rapid.T) testUser {
		return testUser{
			Name: rapid.StringN(1, 20, 100).Draw(t, "name"),
			Age:  rapid.IntRange(0, 130).Draw(t, "age"),
		}
	})
}

// genID generates a non-empty testID.
func genID() *rapid.Generator[testID] {
	return rapid.Custom(func(t *rapid.T) testID {
		return testID(rapid.StringN(1, 20, 100).Draw(t, "id"))
	})
}

// TestProperty_SetGetRoundTrip — after Set(id, val), Get(id) returns val.
func TestProperty_SetGetRoundTrip(t *testing.T) {
	t.Parallel()

	for name, factory := range allTypedStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory()
				defer cleanup()

				ctx := context.Background()
				id := genID().Draw(rt, "id")
				user := genUser().Draw(rt, "user")

				err := store.Set(ctx, id, &user)
				if err != nil {
					rt.Fatalf("Set failed: %v", err)
				}

				got, err := store.Get(ctx, id)
				if err != nil {
					rt.Fatalf("Get failed: %v", err)
				}

				if *got != user {
					rt.Fatalf("round-trip mismatch: set %+v, got %+v", user, *got)
				}
			})
		})
	}
}

// TestProperty_DeleteMakesGetFail — after Delete(id), Get(id) returns ErrNotFound.
func TestProperty_DeleteMakesGetFail(t *testing.T) {
	t.Parallel()

	for name, factory := range allTypedStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory()
				defer cleanup()

				ctx := context.Background()
				id := genID().Draw(rt, "id")
				user := genUser().Draw(rt, "user")

				_ = store.Set(ctx, id, &user)
				_ = store.Delete(ctx, id)

				_, err := store.Get(ctx, id)
				if err == nil {
					rt.Fatalf("Get after Delete should fail, but succeeded")
				}
			})
		})
	}
}

// TestProperty_SetOverwrite — a second Set on the same key replaces the value.
func TestProperty_SetOverwrite(t *testing.T) {
	t.Parallel()

	for name, factory := range allTypedStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory()
				defer cleanup()

				ctx := context.Background()
				id := genID().Draw(rt, "id")
				first := genUser().Draw(rt, "first")
				second := genUser().Draw(rt, "second")

				_ = store.Set(ctx, id, &first)
				_ = store.Set(ctx, id, &second)

				got, err := store.Get(ctx, id)
				if err != nil {
					rt.Fatalf("Get failed: %v", err)
				}

				if *got != second {
					rt.Fatalf("overwrite mismatch: expected %+v, got %+v", second, *got)
				}
			})
		})
	}
}

// TestProperty_HasReflectsExistence — Has returns true after Set, false after Delete.
func TestProperty_HasReflectsExistence(t *testing.T) {
	t.Parallel()

	for name, factory := range allTypedStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory()
				defer cleanup()

				ctx := context.Background()
				id := genID().Draw(rt, "id")
				user := genUser().Draw(rt, "user")

				exists, _ := store.Has(ctx, id)
				if exists {
					rt.Fatalf("Has should be false before Set")
				}

				_ = store.Set(ctx, id, &user)

				exists, _ = store.Has(ctx, id)
				if !exists {
					rt.Fatalf("Has should be true after Set")
				}

				_ = store.Delete(ctx, id)

				exists, _ = store.Has(ctx, id)
				if exists {
					rt.Fatalf("Has should be false after Delete")
				}
			})
		})
	}
}

// TestProperty_CacheInvalidationOnDelete — after Cache.Delete, Get goes back
// to the underlying store (cache miss → store hit, value unchanged).
func TestProperty_CacheInvalidationOnDelete(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		backend := kv.NewMemStore()
		defer backend.Close()

		store := kv.NewTypedStore[testUser, testID](backend)
		cache, err := kv.NewCache[testUser, testID](store)
		if err != nil {
			rt.Fatalf("NewCache failed: %v", err)
		}

		ctx := context.Background()
		id := genID().Draw(rt, "id")
		user := genUser().Draw(rt, "user")

		_ = cache.Set(ctx, id, &user)

		got, err := cache.Get(ctx, id)
		if err != nil || *got != user {
			rt.Fatalf("cache Get after Set: got=%+v err=%v", got, err)
		}

		_ = cache.Delete(ctx, id)

		_, err = cache.Get(ctx, id)
		if err == nil {
			rt.Fatalf("cache Get after Delete should fail")
		}
	})
}

// TestProperty_DistinctKeysIndependent — setting one key doesn't affect another.
func TestProperty_DistinctKeysIndependent(t *testing.T) {
	t.Parallel()

	for name, factory := range allTypedStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory()
				defer cleanup()

				ctx := context.Background()
				id1 := genID().Draw(rt, "id1")
				id2 := genID().Draw(rt, "id2")

				// Only proceed if the keys are actually distinct.
				if id1 == id2 {
					rt.Skip("id1 == id2, not a useful case")
				}

				user1 := genUser().Draw(rt, "user1")
				user2 := genUser().Draw(rt, "user2")

				_ = store.Set(ctx, id1, &user1)
				_ = store.Set(ctx, id2, &user2)
				_ = store.Delete(ctx, id1)

				// id2 must still be present and unchanged.
				got, err := store.Get(ctx, id2)
				if err != nil {
					rt.Fatalf("Get(id2) failed after Delete(id1): %v", err)
				}

				if *got != user2 {
					rt.Fatalf("id2 mutated after Delete(id1): expected %+v, got %+v", user2, *got)
				}
			})
		})
	}
}
