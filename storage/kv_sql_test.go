package storage_test

import (
	"context"
	"errors"
	"testing"

	_ "modernc.org/sqlite" // pure-Go SQLite driver

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func newTestKVStore(t *testing.T) *storage.SQLKVStore {
	t.Helper()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteKVStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteKVStore: %v", err)
	}

	return store
}

func TestSQLKVStore_CRUD(t *testing.T) {
	t.Parallel()

	store := newTestKVStore(t)

	// Get on missing key → ErrNotFound.
	if _, err := store.Get([]byte("missing")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}

	// Has on missing → false.
	has, err := store.Has([]byte("missing"))
	if err != nil || has {
		t.Fatalf("Has missing: has=%v err=%v, want false nil", has, err)
	}

	// Set + Get roundtrip.
	if err := store.Set([]byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != "v1" {
		t.Fatalf("Get: got %q, want %q", got, "v1")
	}

	// Has existing → true.
	has, err = store.Has([]byte("k1"))
	if err != nil || !has {
		t.Fatalf("Has existing: has=%v err=%v, want true nil", has, err)
	}

	// Set overwrite (upsert).
	if err := store.Set([]byte("k1"), []byte("v1-updated")); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}

	got, err = store.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("Get after overwrite: %v", err)
	}

	if string(got) != "v1-updated" {
		t.Fatalf("Get after overwrite: got %q, want %q", got, "v1-updated")
	}

	// Delete.
	if err := store.Delete([]byte("k1")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get([]byte("k1")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}

	// Delete missing is a no-op.
	if err := store.Delete([]byte("never-existed")); err != nil {
		t.Fatalf("Delete missing: err = %v, want nil", err)
	}
}

func TestSQLKVStore_Iterator(t *testing.T) {
	t.Parallel()

	store := newTestKVStore(t)

	// Seed keys across two prefixes plus a bare key, inserted out of order.
	seed := map[string]string{
		"todos:1": "a",
		"todos:2": "b",
		"users:1": "c",
		"plain":   "d",
	}

	for k, v := range seed {
		if err := store.Set([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Set %q: %v", k, err)
		}
	}

	// Empty prefix → all keys in lexicographic order, with matching values.
	allKeys, allVals := collectIter(t, store, nil)
	wantAll := []string{"plain", "todos:1", "todos:2", "users:1"}
	if !equalKeys(allKeys, wantAll) {
		t.Fatalf("iter all: got %v, want %v", allKeys, wantAll)
	}

	wantVals := []string{"d", "a", "b", "c"}
	if !equalKeys(allVals, wantVals) {
		t.Fatalf("iter all values: got %v, want %v", allVals, wantVals)
	}

	// Prefix "todos:" → only todos: keys, ordered.
	todos, _ := collectIter(t, store, []byte("todos:"))
	if !equalKeys(todos, []string{"todos:1", "todos:2"}) {
		t.Fatalf("iter todos: got %v, want [todos:1 todos:2]", todos)
	}

	// Prefix "users:" → single key.
	users, _ := collectIter(t, store, []byte("users:"))
	if !equalKeys(users, []string{"users:1"}) {
		t.Fatalf("iter users: got %v, want [users:1]", users)
	}

	// Prefix matching nothing → empty.
	none, _ := collectIter(t, store, []byte("missing:"))
	if len(none) != 0 {
		t.Fatalf("iter missing: got %v, want empty", none)
	}
}

func TestSQLKVStore_Batch(t *testing.T) {
	t.Parallel()

	store := newTestKVStore(t)

	if err := store.Set([]byte("keep"), []byte("k")); err != nil {
		t.Fatalf("Set keep: %v", err)
	}

	if err := store.Set([]byte("drop"), []byte("d")); err != nil {
		t.Fatalf("Set drop: %v", err)
	}

	// Committed batch: add one, delete one, atomically.
	batch, err := store.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	if err := batch.Set([]byte("added"), []byte("n")); err != nil {
		t.Fatalf("batch Set: %v", err)
	}

	if err := batch.Delete([]byte("drop")); err != nil {
		t.Fatalf("batch Delete: %v", err)
	}

	if err := batch.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if got, err := store.Get([]byte("added")); err != nil || string(got) != "n" {
		t.Fatalf("after commit Get added: got %q err=%v, want n", got, err)
	}

	if _, err := store.Get([]byte("drop")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("after commit Get drop: err=%v, want ErrNotFound", err)
	}

	// Discarded batch (Close without Commit) writes nothing.
	discard, err := store.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	if err := discard.Set([]byte("ephemeral"), []byte("x")); err != nil {
		t.Fatalf("discard Set: %v", err)
	}

	if err := discard.Close(); err != nil {
		t.Fatalf("discard Close: %v", err)
	}

	if _, err := store.Get([]byte("ephemeral")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("after discard Get ephemeral: err=%v, want ErrNotFound", err)
	}

	// Commit after Close is a no-op (no panic, no double-commit error).
	if err := discard.Commit(); err != nil {
		t.Fatalf("Commit after Close: %v, want nil", err)
	}
}

func collectIter(t *testing.T, store *storage.SQLKVStore, prefix []byte) ([]string, []string) {
	t.Helper()

	iter, err := store.NewIterator(prefix)
	if err != nil {
		t.Fatalf("NewIterator %q: %v", prefix, err)
	}

	defer func() { _ = iter.Close() }()

	var keys, vals []string

	for iter.Next() {
		keys = append(keys, string(iter.Key()))
		vals = append(vals, string(iter.Value()))
	}

	if err := iter.Error(); err != nil {
		t.Fatalf("iter Error: %v", err)
	}

	return keys, vals
}

func equalKeys(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
