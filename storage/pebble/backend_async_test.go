package pebble

import (
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"
)

func openFlagTestBackend(t *testing.T, opts ...BackendOption) *Backend {
	t.Helper()

	database, err := pebble.Open(t.TempDir(), &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open failed: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	backend, err := NewBackend(database, slog.Default(), opts...)
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}

	return backend
}

func readModelAdapter(t *testing.T, backend *Backend) *KVAdapter {
	t.Helper()

	adapter, ok := backend.readMods.(*KVAdapter)
	if !ok {
		t.Fatalf("Backend read models should be a *KVAdapter, got %T", backend.readMods)
	}

	return adapter
}

func TestBackend_DefaultAllStoresSyncWrites(t *testing.T) {
	t.Parallel()

	backend := openFlagTestBackend(t)

	for name, store := range map[string]*storeBase{
		"events":     &backend.events.storeBase,
		"commands":   &backend.commands.storeBase,
		"queries":    &backend.queries.storeBase,
		"snapshot":   &backend.snapshot.storeBase,
		"checkpoint": &backend.checkpt.storeBase,
	} {
		if !store.syncWrites {
			t.Errorf("%s store should default to sync writes", name)
		}
	}

	if adapter := readModelAdapter(t, backend); !adapter.syncWrites {
		t.Error("backend read-model KV store should stay synchronous by default")
	}
}

func TestBackend_WithBackendAsyncWritesFlipsEveryStore(t *testing.T) {
	t.Parallel()

	backend := openFlagTestBackend(t, WithBackendAsyncWrites())

	for name, store := range map[string]*storeBase{
		"events":     &backend.events.storeBase,
		"commands":   &backend.commands.storeBase,
		"queries":    &backend.queries.storeBase,
		"snapshot":   &backend.snapshot.storeBase,
		"checkpoint": &backend.checkpt.storeBase,
	} {
		if store.syncWrites {
			t.Errorf("%s store should be async with WithBackendAsyncWrites", name)
		}

		if store.writeOptions() != nil {
			t.Errorf("%s store writeOptions() should be nil when async", name)
		}
	}

	if adapter := readModelAdapter(t, backend); adapter.syncWrites {
		t.Error("backend read-model KV store should be async with WithBackendAsyncWrites")
	}
}
