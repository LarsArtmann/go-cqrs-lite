package kv

import (
	"bytes"
	"errors"
	"testing"
)

// --- Get / Set / Delete / Has ---

func TestMemStore_SetAndGet(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	key := []byte("user:1")
	val := []byte(`{"name":"alice"}`)

	if err := s.Set(key, val); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !bytes.Equal(got, val) {
		t.Fatalf("Get = %q, want %q", got, val)
	}
}

func TestMemStore_GetNotFound(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	_, err := s.Get([]byte("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_Has(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	if err := s.Set([]byte("k1"), []byte("v1")); err != nil {
		t.Fatal(err)
	}

	exists, err := s.Has([]byte("k1"))
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("Has(k1) = false, want true")
	}

	exists, err = s.Has([]byte("missing"))
	if err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("Has(missing) = true, want false")
	}
}

func TestMemStore_Delete(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	key := []byte("k1")

	if err := s.Set(key, []byte("v1")); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(key)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_DeleteMissing(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	if err := s.Delete([]byte("never-existed")); err != nil {
		t.Fatalf("Delete missing should be no-op, got: %v", err)
	}
}

// --- Defensive cloning ---

func TestMemStore_GetReturnsClone(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	original := []byte("value")
	_ = s.Set([]byte("k"), original)

	got, _ := s.Get([]byte("k"))
	got[0] = 'X'

	again, _ := s.Get([]byte("k"))
	if !bytes.Equal(again, original) {
		t.Fatalf("Get returned a reference, not a clone: got %q, want %q", again, original)
	}
}

func TestMemStore_SetClonesValue(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	val := []byte("value")
	_ = s.Set([]byte("k"), val)
	val[0] = 'X'

	got, _ := s.Get([]byte("k"))
	if got[0] != 'v' {
		t.Fatalf("Set did not clone: got %q, want %q", got, "value")
	}
}

// --- Closed store ---

func TestMemStore_CloseBlocksOps(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	_ = s.Close()

	want := ErrClosed

	if _, err := s.Get([]byte("k")); !errors.Is(err, want) {
		t.Fatalf("Get after close: %v, want %v", err, want)
	}

	if _, err := s.Has([]byte("k")); !errors.Is(err, want) {
		t.Fatalf("Has after close: %v, want %v", err, want)
	}

	if err := s.Set([]byte("k"), []byte("v")); !errors.Is(err, want) {
		t.Fatalf("Set after close: %v, want %v", err, want)
	}

	if err := s.Delete([]byte("k")); !errors.Is(err, want) {
		t.Fatalf("Delete after close: %v, want %v", err, want)
	}

	if _, err := s.Batch(); !errors.Is(err, want) {
		t.Fatalf("Batch after close: %v, want %v", err, want)
	}

	if _, err := s.NewIterator(nil); !errors.Is(err, want) {
		t.Fatalf("NewIterator after close: %v, want %v", err, want)
	}
}

// --- Iterator ---

func TestMemStore_IteratorOrdering(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	keys := []string{"c", "a", "b", "e", "d"}
	for _, k := range keys {
		_ = s.Set([]byte(k), []byte("val-"+k))
	}

	iter, err := s.NewIterator(nil)
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer iter.Close()

	var got []string

	for iter.Next() {
		got = append(got, string(iter.Key()))
	}

	want := []string{"a", "b", "c", "d", "e"}

	if len(got) != len(want) {
		t.Fatalf("iterator returned %d keys, want %d", len(got), len(want))
	}

	for i, k := range got {
		if k != want[i] {
			t.Errorf("iterator[%d] = %q, want %q", i, k, want[i])
		}
	}
}

func TestMemStore_IteratorPrefix(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	_ = s.Set([]byte("user:1"), []byte("alice"))
	_ = s.Set([]byte("user:2"), []byte("bob"))
	_ = s.Set([]byte("order:1"), []byte("shoes"))

	iter, err := s.NewIterator([]byte("user:"))
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer iter.Close()

	var got []string

	for iter.Next() {
		got = append(got, string(iter.Key()))
	}

	if len(got) != 2 {
		t.Fatalf("prefix iterator returned %d keys, want 2", len(got))
	}

	if string(got[0]) != "user:1" {
		t.Errorf("first key = %q, want %q", got[0], "user:1")
	}

	if string(got[1]) != "user:2" {
		t.Errorf("second key = %q, want %q", got[1], "user:2")
	}
}

func TestMemStore_IteratorValue(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	_ = s.Set([]byte("k1"), []byte("val1"))

	iter, _ := s.NewIterator(nil)
	defer iter.Close()

	if !iter.Next() {
		t.Fatal("Next = false, want true")
	}

	if string(iter.Value()) != "val1" {
		t.Errorf("Value = %q, want %q", iter.Value(), "val1")
	}

	if iter.Error() != nil {
		t.Errorf("Error = %v, want nil", iter.Error())
	}
}

func TestMemStore_IteratorEmpty(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	iter, err := s.NewIterator(nil)
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer iter.Close()

	if iter.Next() {
		t.Fatal("Next on empty store = true, want false")
	}
}

// --- Batch ---

func TestMemStore_BatchCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	_ = s.Set([]byte("existing"), []byte("old"))

	batch, err := s.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Set([]byte("b"), []byte("2"))
	_ = batch.Delete([]byte("existing"))

	if err := batch.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	val, _ := s.Get([]byte("a"))
	if string(val) != "1" {
		t.Errorf("batch Set a: got %q, want %q", val, "1")
	}

	val, _ = s.Get([]byte("b"))
	if string(val) != "2" {
		t.Errorf("batch Set b: got %q, want %q", val, "2")
	}

	_, err = s.Get([]byte("existing"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("batch Delete existing: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchCloseDiscards(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	batch, _ := s.Batch()
	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Close()

	if err := batch.Commit(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit after Close: %v, want ErrClosed", err)
	}

	_, err := s.Get([]byte("a"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after discarded batch: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchAfterCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer s.Close()

	batch, _ := s.Batch()
	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Commit()

	if err := batch.Set([]byte("b"), []byte("2")); !errors.Is(err, ErrClosed) {
		t.Fatalf("Set after Commit: %v, want ErrClosed", err)
	}
}

// --- Interface conformance ---

func TestMemStore_ImplementsStore(t *testing.T) {
	t.Parallel()

	var _ Store = (*MemStore)(nil)
	var _ Reader = (*MemStore)(nil)
	var _ Writer = (*MemStore)(nil)
	var _ Iterator = (*memIterator)(nil)
	var _ Batch = (*memBatch)(nil)
}
