package kvstore_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// faultBackend wraps a real MemStore and can inject errors on specific operations.
type faultBackend struct {
	*kv.MemStore
	getErr     error
	setIAErr   error
	setErr     error
	closeErr   error
	closeCalls int
}

func (f *faultBackend) Get(ctx context.Context, key []byte) ([]byte, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.MemStore.Get(ctx, key)
}

func (f *faultBackend) Set(ctx context.Context, key, val []byte) error {
	if f.setErr != nil {
		return f.setErr
	}
	return f.MemStore.Set(ctx, key, val)
}

func (f *faultBackend) SetIfAbsent(ctx context.Context, key, val []byte) (bool, error) {
	if f.setIAErr != nil {
		return false, f.setIAErr
	}
	return f.MemStore.SetIfAbsent(ctx, key, val)
}

func (f *faultBackend) Close() error {
	f.closeCalls++
	if f.closeErr != nil {
		return f.closeErr
	}
	return f.MemStore.Close()
}

func TestStore_Seen_CorruptedValue(t *testing.T) {
	t.Parallel()
	backend := kv.NewMemStore()
	store := kvstore.New(backend)
	ctx := context.Background()
	_ = backend.Set(ctx, []byte("bad"), []byte("not-a-number"))
	seen, err := store.Seen(ctx, "bad")
	if seen {
		t.Fatal("expected seen=false for corrupted value")
	}
	var ferr errorfamily.Error
	if !errors.As(err, &ferr) {
		t.Fatalf("expected errorfamily.Error, got %T: %v", err, err)
	}
}

func TestStore_Seen_BackendError(t *testing.T) {
	t.Parallel()
	fb := &faultBackend{
		MemStore: kv.NewMemStore(),
		getErr:   errors.New("backend down"),
	}
	store := kvstore.New(fb)
	_, err := store.Seen(context.Background(), "key")
	if err == nil {
		t.Fatal("expected error from backend failure")
	}
}

func TestStore_Record_SetIfAbsentError(t *testing.T) {
	t.Parallel()
	fb := &faultBackend{
		MemStore: kv.NewMemStore(),
		setIAErr: errors.New("write failed"),
	}
	store := kvstore.New(fb)
	err := store.Record(context.Background(), "key", time.Minute)
	if err == nil {
		t.Fatal("expected error from SetIfAbsent failure")
	}
}

func TestStore_CheckAndRecord_Expired_OverwritesAndClaims(t *testing.T) {
	t.Parallel()
	backend := kv.NewMemStore()
	store := kvstore.New(backend)
	ctx := context.Background()
	_ = store.Record(ctx, "key", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	if err := store.CheckAndRecord(ctx, "key", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord on expired key: %v", err)
	}
	seen, err := store.Seen(ctx, "key")
	if err != nil {
		t.Fatalf("Seen after overwrite: %v", err)
	}
	if !seen {
		t.Fatal("expected key to be seen after CheckAndRecord overwrite")
	}
}

func TestStore_CheckAndRecord_BackendError(t *testing.T) {
	t.Parallel()
	fb := &faultBackend{
		MemStore:  kv.NewMemStore(),
		setIAErr:  errors.New("backend down"),
	}
	store := kvstore.New(fb)
	err := store.CheckAndRecord(context.Background(), "key", time.Minute)
	if err == nil {
		t.Fatal("expected error from SetIfAbsent failure")
	}
}

func TestStore_CheckAndRecord_CorruptedExisting(t *testing.T) {
	t.Parallel()
	backend := kv.NewMemStore()
	store := kvstore.New(backend)
	ctx := context.Background()
	_ = backend.Set(ctx, []byte("key"), []byte("corrupt"))
	err := store.CheckAndRecord(ctx, "key", time.Minute)
	if err == nil {
		t.Fatal("expected corruption error")
	}
	var ferr errorfamily.Error
	if !errors.As(err, &ferr) {
		t.Fatalf("expected errorfamily.Error, got %T: %v", err, err)
	}
}

func TestStore_CheckAndRecord_RetryOnRace(t *testing.T) {
	t.Parallel()
	backend := kv.NewMemStore()
	store := kvstore.New(backend)
	ctx := context.Background()
	if err := store.CheckAndRecord(ctx, "key", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}
	err := store.CheckAndRecord(ctx, "key", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestStore_Close_PassesThrough(t *testing.T) {
	t.Parallel()
	fb := &faultBackend{MemStore: kv.NewMemStore()}
	store := kvstore.New(fb)
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if fb.closeCalls != 1 {
		t.Fatalf("expected 1 Close call on backend, got %d", fb.closeCalls)
	}
}

func TestStore_Close_BackendError(t *testing.T) {
	t.Parallel()
	fb := &faultBackend{
		MemStore:  kv.NewMemStore(),
		closeErr:  io.ErrClosedPipe,
	}
	store := kvstore.New(fb)
	if err := store.Close(); err == nil {
		t.Fatal("expected error from backend Close failure")
	}
}
