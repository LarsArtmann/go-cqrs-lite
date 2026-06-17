package pebble

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

func openTestKVStore(t *testing.T, opts ...KVOption) kv.Store {
	t.Helper()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	// Use WithBorrowedDB so the adapter won't close the DB; t.Cleanup owns the lifecycle.
	opts = append([]KVOption{WithBorrowedDB()}, opts...)

	t.Cleanup(func() { _ = database.Close() })

	return NewKVStore(database, opts...)
}

// ── CRUD tests ───────────────────────────────────────────────

func TestKVAdapter_SetAndGet(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	err := store.Set([]byte("k1"), []byte("v1"))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := store.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(val) != "v1" {
		t.Fatalf("got %q, want %q", val, "v1")
	}
}

func TestKVAdapter_GetNotFound(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	_, err := store.Get([]byte("missing"))
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("got err %v, want kv.ErrNotFound", err)
	}
}

func TestKVAdapter_GetReturnsCopy(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	original := []byte("value")
	err := store.Set([]byte("k"), original)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get([]byte("k"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	got[0] = 'X'

	again, err := store.Get([]byte("k"))
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}

	if string(again) != "value" {
		t.Fatalf("value mutated: got %q, want %q", again, "value")
	}
}

func TestKVAdapter_Has(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	err := store.Set([]byte("exists"), []byte("v"))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	has, err := store.Has([]byte("exists"))
	if err != nil {
		t.Fatalf("Has(exists): %v", err)
	}

	if !has {
		t.Fatal("Has(exists) = false, want true")
	}

	has, err = store.Has([]byte("missing"))
	if err != nil {
		t.Fatalf("Has(missing): %v", err)
	}

	if has {
		t.Fatal("Has(missing) = true, want false")
	}
}

func TestKVAdapter_Delete(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

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
		t.Fatalf("after delete: got err %v, want kv.ErrNotFound", err)
	}
}

func TestKVAdapter_DeleteNonExistent(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	err := store.Delete([]byte("never-existed"))
	if err != nil {
		t.Fatalf("Delete non-existent should be no-op, got: %v", err)
	}
}

// ── Iterator tests ───────────────────────────────────────────

func TestKVIterator_AllKeys(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	keys := []string{"a", "b", "c", "d"}
	for _, k := range keys {
		err := store.Set([]byte(k), []byte("val-"+k))
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

	want := keys // already sorted
	if len(got) != len(want) {
		t.Fatalf("got %d keys, want %d", len(got), len(want))
	}

	for i, k := range got {
		if k != want[i] {
			t.Fatalf("key[%d] = %q, want %q", i, k, want[i])
		}
	}
}

func TestKVIterator_PrefixFilter(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	pairs := map[string]string{
		"prefix:a": "1",
		"prefix:b": "2",
		"prefix:c": "3",
		"other:1":  "x",
		"other:2":  "y",
	}

	for k, v := range pairs {
		err := store.Set([]byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set %s: %v", k, err)
		}
	}

	iter, err := store.NewIterator([]byte("prefix:"))
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

	want := []string{"prefix:a", "prefix:b", "prefix:c"}
	if len(got) != len(want) {
		t.Fatalf("got %d keys, want %d (%v)", len(got), len(want), got)
	}

	for i, k := range got {
		if k != want[i] {
			t.Fatalf("key[%d] = %q, want %q", i, k, want[i])
		}
	}
}

func TestKVIterator_ValuesAreSafe(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	err := store.Set([]byte("k"), []byte("original"))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	iter, err := store.NewIterator(nil)
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}
	defer func() { _ = iter.Close() }()

	if !iter.Next() {
		t.Fatal("iter.Next = false")
	}

	val := iter.Value()
	val[0] = 'X'

	again, err := store.Get([]byte("k"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(again) != "original" {
		t.Fatalf("value mutated via iterator: got %q, want %q", again, "original")
	}
}

// ── Batch tests ──────────────────────────────────────────────

func TestKVBatch_AtomicCommit(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	batch, err := store.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	err = batch.Set([]byte("batch:1"), []byte("a"))
	if err != nil {
		t.Fatalf("batch.Set: %v", err)
	}

	err = batch.Set([]byte("batch:2"), []byte("b"))
	if err != nil {
		t.Fatalf("batch.Set: %v", err)
	}

	err = batch.Delete([]byte("batch:1"))
	if err != nil {
		t.Fatalf("batch.Delete: %v", err)
	}

	// Before commit, keys should not be visible.
	_, err = store.Get([]byte("batch:2"))
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("before commit: got err %v, want kv.ErrNotFound", err)
	}

	err = batch.Commit()
	if err != nil {
		t.Fatalf("batch.Commit: %v", err)
	}

	// After commit, only batch:2 should exist (batch:1 was set then deleted).
	_, err = store.Get([]byte("batch:1"))
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("after commit: batch:1 should be deleted, got err %v", err)
	}

	val, err := store.Get([]byte("batch:2"))
	if err != nil {
		t.Fatalf("after commit: Get batch:2: %v", err)
	}

	if string(val) != "b" {
		t.Fatalf("after commit: batch:2 = %q, want %q", val, "b")
	}
}

func TestKVBatch_CloseDiscards(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

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
		t.Fatalf("after Close: got err %v, want kv.ErrNotFound", err)
	}
}

func TestKVBatch_CommitThenCloseIsNoOp(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	batch, err := store.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	err = batch.Set([]byte("k"), []byte("v"))
	if err != nil {
		t.Fatalf("batch.Set: %v", err)
	}

	err = batch.Commit()
	if err != nil {
		t.Fatalf("batch.Commit: %v", err)
	}

	err = batch.Close()
	if err != nil {
		t.Fatalf("Close after Commit should be no-op: %v", err)
	}
}

// ── Close tests ──────────────────────────────────────────────

func TestKVAdapter_CloseOwnedDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	store := NewKVStore(database)

	err = store.Set([]byte("k"), []byte("v"))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, operations should return ErrClosed.
	_, err = store.Get([]byte("k"))
	if !errors.Is(err, kv.ErrClosed) {
		t.Fatalf("after Close: got err %v, want kv.ErrClosed", err)
	}
}

func TestKVAdapter_CloseBorrowedDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}
	defer func() { _ = database.Close() }()

	store := NewKVStore(database, WithBorrowedDB())

	err = store.Set([]byte("k"), []byte("v"))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Borrowed DB should still work directly after adapter Close.
	val, _, err := database.Get([]byte("k"))
	if err != nil {
		t.Fatalf("db.Get after adapter Close: %v", err)
	}

	if string(val) != "v" {
		t.Fatalf("got %q, want %q", val, "v")
	}
}

func TestKVAdapter_DoubleCloseIsSafe(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t)

	err := store.Close()
	if err != nil {
		t.Fatalf("first Close: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// ── prefixUpperBound tests ────────────────────────────────────

func TestPrefixUpperBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix string
		want   string
	}{
		{"abc", "abd"},
		{"a", "b"},
		{"\x00", "\x01"},
		{"a\xff", "b"}, // overflow: last byte 0xff, previous increments
	}

	for _, tt := range tests {
		got := prefixUpperBound([]byte(tt.prefix))
		if !bytes.Equal(got, []byte(tt.want)) {
			t.Fatalf(
				"prefixUpperBound(%q) = %q (%v), want %q (%v)",
				tt.prefix, got, got, tt.want, []byte(tt.want),
			)
		}
	}
}

// ── compile-time interface check ─────────────────────────────

func TestKVAdapter_ImplementsKVStore(t *testing.T) {
	t.Parallel()

	var _ kv.Store = (*KVAdapter)(nil)
	var _ kv.Iterator = (*pebbleIterator)(nil)
	var _ kv.Batch = (*pebbleBatch)(nil)
}

// ── Stress: many keys with iterator ──────────────────────────

func TestKVIterator_ManyKeys(t *testing.T) {
	t.Parallel()

	store := openTestKVStore(t, WithKVSyncWrites())

	const n = 500

	for i := range n {
		err := store.Set(fmt.Appendf(nil, "key:%05d", i), []byte("val"))
		if err != nil {
			t.Fatalf("Set %d: %v", i, err)
		}
	}

	iter, err := store.NewIterator([]byte("key:"))
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}
	defer func() { _ = iter.Close() }()

	count := 0

	for iter.Next() {
		count++
	}

	if iter.Error() != nil {
		t.Fatalf("iter.Error: %v", iter.Error())
	}

	if count != n {
		t.Fatalf("iterated %d keys, want %d", count, n)
	}
}
