package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// TestSQLBackend_KVStore proves the backend hands out one working kv.Store per
// backend and that it persists through set/get/delete like any other store.
func TestSQLBackend_KVStore(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	backend, err := storage.NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}
	t.Cleanup(func() { _ = backend.Close() })

	store, err := backend.KVStore()
	if err != nil {
		t.Fatalf("KVStore: %v", err)
	}

	again, err := backend.KVStore()
	if err != nil {
		t.Fatalf("second KVStore: %v", err)
	}

	if store != again {
		t.Fatal("KVStore returned a different store on the second call")
	}

	ctx := context.Background()

	if err := store.Set(ctx, []byte("task:1"), []byte(`{"title":"ship"}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, []byte("task:1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != `{"title":"ship"}` {
		t.Errorf("Get = %q, want the stored value", got)
	}

	has, err := store.Has(ctx, []byte("task:1"))
	if err != nil {
		t.Fatalf("Has: %v", err)
	}

	if !has {
		t.Error("Has = false for a key that was just set")
	}

	if err := store.Delete(ctx, []byte("task:1")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(ctx, []byte("task:1")); !errors.Is(err, kv.ErrNotFound) {
		t.Errorf("Get after Delete = %v, want kv.ErrNotFound", err)
	}
}
