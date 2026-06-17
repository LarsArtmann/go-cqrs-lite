package kv

import (
	"bytes"
	"errors"
	"sync"
	"testing"
)

// --- Get / Set / Delete / Has ---

func TestMemStore_SetAndGet(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	key := []byte("user:1")
	val := []byte(`{"name":"alice"}`)

	err := s.Set(key, val)
	if err != nil {
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
	defer func() { _ = s.Close() }()

	_, err := s.Get([]byte("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_Has(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	err := s.Set([]byte("k1"), []byte("v1"))
	if err != nil {
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
	defer func() { _ = s.Close() }()

	key := []byte("k1")

	err := s.Set(key, []byte("v1"))
	if err != nil {
		t.Fatal(err)
	}

	err = s.Delete(key)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = s.Get(key)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_DeleteMissing(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	err := s.Delete([]byte("never-existed"))
	if err != nil {
		t.Fatalf("Delete missing should be no-op, got: %v", err)
	}
}

// --- Defensive cloning ---

func TestMemStore_GetReturnsClone(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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

	_, err := s.Get([]byte("k"))
	if !errors.Is(err, want) {
		t.Fatalf("Get after close: %v, want %v", err, want)
	}

	_, err = s.Has([]byte("k"))
	if !errors.Is(err, want) {
		t.Fatalf("Has after close: %v, want %v", err, want)
	}

	err = s.Set([]byte("k"), []byte("v"))
	if !errors.Is(err, want) {
		t.Fatalf("Set after close: %v, want %v", err, want)
	}

	err = s.Delete([]byte("k"))
	if !errors.Is(err, want) {
		t.Fatalf("Delete after close: %v, want %v", err, want)
	}

	_, err = s.Batch()
	if !errors.Is(err, want) {
		t.Fatalf("Batch after close: %v, want %v", err, want)
	}

	_, err = s.NewIterator(nil)
	if !errors.Is(err, want) {
		t.Fatalf("NewIterator after close: %v, want %v", err, want)
	}
}

// --- Concurrent access ---

func TestMemStore_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("init"), []byte("0"))

	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := []byte("key")
			_ = s.Set(key, []byte("val"))
			_, _ = s.Get(key)
			_, _ = s.Has(key)
		}(i)
	}

	wg.Wait()

	val, err := s.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get after concurrent writes: %v", err)
	}

	if string(val) != "val" {
		t.Errorf("concurrent Set produced wrong value: got %q", val)
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
