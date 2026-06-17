package kv

import (
	"testing"
)

func TestMemStore_IteratorOrdering(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	keys := []string{"c", "a", "b", "e", "d"}
	for _, k := range keys {
		_ = s.Set([]byte(k), []byte("val-"+k))
	}

	iter, err := s.NewIterator(nil)
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer func() { _ = iter.Close() }()

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
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("user:1"), []byte("alice"))
	_ = s.Set([]byte("user:2"), []byte("bob"))
	_ = s.Set([]byte("order:1"), []byte("shoes"))

	iter, err := s.NewIterator([]byte("user:"))
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer func() { _ = iter.Close() }()

	var got []string

	for iter.Next() {
		got = append(got, string(iter.Key()))
	}

	if len(got) != 2 {
		t.Fatalf("prefix iterator returned %d keys, want 2", len(got))
	}

	if got[0] != "user:1" {
		t.Errorf("first key = %q, want %q", got[0], "user:1")
	}

	if got[1] != "user:2" {
		t.Errorf("second key = %q, want %q", got[1], "user:2")
	}
}

func TestMemStore_IteratorValue(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("k1"), []byte("val1"))

	iter, _ := s.NewIterator(nil)
	defer func() { _ = iter.Close() }()

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
	defer func() { _ = s.Close() }()

	iter, err := s.NewIterator(nil)
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	defer func() { _ = iter.Close() }()

	if iter.Next() {
		t.Fatal("Next on empty store = true, want false")
	}
}

func TestMemStore_IteratorSnapshot(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("a"), []byte("1"))
	_ = s.Set([]byte("b"), []byte("2"))

	iter, _ := s.NewIterator(nil)
	defer func() { _ = iter.Close() }()

	_ = s.Set([]byte("c"), []byte("3"))
	_ = s.Delete([]byte("a"))

	var count int

	for iter.Next() {
		count++
	}

	if count != 2 {
		t.Fatalf("snapshot iterator saw %d keys, want 2 (snapshot taken before mutations)", count)
	}
}
