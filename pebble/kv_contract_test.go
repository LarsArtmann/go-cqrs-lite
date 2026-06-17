package pebble

import (
	"errors"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

// runKVStoreContract runs a standardized contract test suite against any kv.Store.
// This proves interface conformance — the same semantics must hold regardless of
// the underlying implementation (MemStore, PebbleAdapter, future backends).
func runKVStoreContract(t *testing.T, makeStore func(t *testing.T) kv.Store) {
	t.Helper()

	t.Run("GetSet", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)
		err := store.Set([]byte("k"), []byte("v"))
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		val, err := store.Get([]byte("k"))
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if string(val) != "v" {
			t.Fatalf("got %q, want %q", val, "v")
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)
		_, err := store.Get([]byte("missing"))
		if !errors.Is(err, kv.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("Has", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		err := store.Set([]byte("exists"), []byte("v"))
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		has, err := store.Has([]byte("exists"))
		if err != nil || !has {
			t.Fatalf("Has(exists) = %v, %v, want true, nil", has, err)
		}

		has, err = store.Has([]byte("missing"))
		if err != nil || has {
			t.Fatalf("Has(missing) = %v, %v, want false, nil", has, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		err := store.Set([]byte("k"), []byte("v"))
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		err = store.Delete([]byte("k"))
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = store.Get([]byte("k"))
		if !errors.Is(err, kv.ErrNotFound) {
			t.Fatalf("after Delete: got %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)
		err := store.Delete([]byte("never"))
		if err != nil {
			t.Fatalf("Delete non-existent should be no-op: %v", err)
		}
	})

	t.Run("BatchAtomic", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		batch, err := store.Batch()
		if err != nil {
			t.Fatalf("Batch: %v", err)
		}

		err = batch.Set([]byte("batch:1"), []byte("a"))
		if err != nil {
			t.Fatalf("batch.Set: %v", err)
		}

		err = batch.Commit()
		if err != nil {
			t.Fatalf("batch.Commit: %v", err)
		}

		val, err := store.Get([]byte("batch:1"))
		if err != nil {
			t.Fatalf("Get after commit: %v", err)
		}

		if string(val) != "a" {
			t.Fatalf("got %q, want %q", val, "a")
		}
	})

	t.Run("BatchCloseDiscards", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		batch, err := store.Batch()
		if err != nil {
			t.Fatalf("Batch: %v", err)
		}

		err = batch.Set([]byte("discarded"), []byte("v"))
		if err != nil {
			t.Fatalf("batch.Set: %v", err)
		}

		err = batch.Close()
		if err != nil {
			t.Fatalf("batch.Close: %v", err)
		}

		_, err = store.Get([]byte("discarded"))
		if !errors.Is(err, kv.ErrNotFound) {
			t.Fatalf("after Close: got %v, want ErrNotFound", err)
		}
	})

	t.Run("Iterator", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		for _, k := range []string{"a", "b", "c"} {
			err := store.Set([]byte(k), []byte("val"))
			if err != nil {
				t.Fatalf("Set %s: %v", k, err)
			}
		}

		iter, err := store.NewIterator(nil)
		if err != nil {
			t.Fatalf("NewIterator: %v", err)
		}
		defer func() { _ = iter.Close() }()

		var got []string
		for iter.Next() {
			got = append(got, string(iter.Key()))
		}

		if iter.Error() != nil {
			t.Fatalf("iter.Error: %v", iter.Error())
		}

		want := []string{"a", "b", "c"}
		if len(got) != len(want) {
			t.Fatalf("got %d keys, want %d (%v)", len(got), len(want), got)
		}

		for i, k := range got {
			if k != want[i] {
				t.Fatalf("key[%d] = %q, want %q", i, k, want[i])
			}
		}
	})

	t.Run("IteratorPrefix", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		for _, k := range []string{"pfx:1", "pfx:2", "other:1"} {
			err := store.Set([]byte(k), []byte("val"))
			if err != nil {
				t.Fatalf("Set %s: %v", k, err)
			}
		}

		iter, err := store.NewIterator([]byte("pfx:"))
		if err != nil {
			t.Fatalf("NewIterator: %v", err)
		}
		defer func() { _ = iter.Close() }()

		var got []string
		for iter.Next() {
			got = append(got, string(iter.Key()))
		}

		want := []string{"pfx:1", "pfx:2"}
		if len(got) != len(want) {
			t.Fatalf("got %d keys, want %d (%v)", len(got), len(want), got)
		}
	})

	t.Run("ValueSafety", func(t *testing.T) {
		t.Parallel()

		store := makeStore(t)

		err := store.Set([]byte("k"), []byte("original"))
		if err != nil {
			t.Fatalf("Set: %v", err)
		}

		val, err := store.Get([]byte("k"))
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		val[0] = 'X'

		again, err := store.Get([]byte("k"))
		if err != nil {
			t.Fatalf("Get again: %v", err)
		}

		if string(again) != "original" {
			t.Fatalf("value mutated: got %q, want %q", again, "original")
		}
	})
}

// ── Run contract against Pebble adapter ──────────────────────

func TestKVContract_PebbleAdapter(t *testing.T) {
	t.Parallel()

	runKVStoreContract(t, func(t *testing.T) kv.Store {
		return openTestKVStore(t)
	})
}

// ── Run contract against MemStore for comparison ─────────────

func TestKVContract_MemStore(t *testing.T) {
	t.Parallel()

	// This test lives in the pebble package to verify that the Pebble
	// adapter produces semantically identical results to the reference
	// MemStore implementation. It's a local import — no circular dependency.
	runKVStoreContract(t, func(t *testing.T) kv.Store {
		store := kv.NewMemStore()
		t.Cleanup(func() { _ = store.Close() })

		return store
	})
}

// Suppress unused import warning when tests are compiled without the
// pebble.Open path above.
var _ = pebble.Options{}
