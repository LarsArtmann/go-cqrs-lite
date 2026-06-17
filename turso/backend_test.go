package turso_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
	"github.com/larsartmann/go-cqrs-lite/turso/v2"
)

func newBackendDB(t *testing.T) (*storage.SQLBackend, func()) {
	t.Helper()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	backend, err := turso.NewBackend(db)
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	return backend, func() {
		_ = backend.Close()
		_ = db.Close()
	}
}

func TestNewBackend(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestBackend_EventStore(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	store := backend.EventStore()
	if store == nil {
		t.Fatal("expected non-nil event store")
	}

	// Verify it actually works.
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.created", aggID, "TestAggregate", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	ref := event.NewAggregateRef("TestAggregate", aggID)
	ctx := context.Background()
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestBackend_CommandStore(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	ctx := context.Background()
	cmdStore, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore: %v", err)
	}

	if cmdStore == nil {
		t.Fatal("expected non-nil command store")
	}

	aggID := id.NewAggregateID()
	cmdRef := command.NewAggregateRef("User", aggID)
	cmd, err := command.NewPersistedCommand("CreateUser", cmdRef, []byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := cmdStore.Save(ctx, cmdRef, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := cmdStore.Load(ctx, cmdRef)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}
}

func TestBackend_QueryStore(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	ctx := context.Background()
	qStore, err := backend.QueryStore()
	if err != nil {
		t.Fatalf("QueryStore: %v", err)
	}

	if qStore == nil {
		t.Fatal("expected non-nil query store")
	}

	pq, err := query.NewPersistedQuery("user.search", []byte(`{"q":"alice"}`))
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	if err := qStore.SaveQuery(ctx, pq); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}
}

func TestBackend_SnapshotStore(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	snapStore, err := backend.SnapshotStore()
	if err != nil {
		t.Fatalf("SnapshotStore: %v", err)
	}

	if snapStore == nil {
		t.Fatal("expected non-nil snapshot store")
	}
}

func TestBackend_CheckpointStore(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	cpStore, err := backend.CheckpointStore()
	if err != nil {
		t.Fatalf("CheckpointStore: %v", err)
	}

	if cpStore == nil {
		t.Fatal("expected non-nil checkpoint store")
	}
}

func TestBackend_LazyInit_SameInstance(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	// Calling CommandStore twice must return the same instance.
	first, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore (1st): %v", err)
	}

	second, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore (2nd): %v", err)
	}

	if first != second {
		t.Error("expected CommandStore() to return the same instance on repeated calls")
	}
}

func TestBackend_LazyInit_Concurrent(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	defer cleanup()

	const goroutines = 20
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[*storage.SQLCommandStore]bool)
		errs []error
	)

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			store, err := backend.CommandStore()
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()

				return
			}

			mu.Lock()
			seen[store] = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}

	if len(seen) != 1 {
		t.Errorf("expected 1 unique command store instance, got %d", len(seen))
	}
}

func TestBackend_Close(t *testing.T) {
	t.Parallel()

	backend, _ := newBackendDB(t)

	// Materialize all stores so Close has something to close.
	_, _ = backend.CommandStore()
	_, _ = backend.QueryStore()
	_, _ = backend.SnapshotStore()
	_, _ = backend.CheckpointStore()

	if err := backend.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After Close, the event store should report closed (Infrastructure family).
	ctx := context.Background()
	aggID := id.NewAggregateID()
	_, err := backend.EventStore().Load(ctx, event.NewAggregateRef("X", aggID))
	if err == nil {
		t.Fatal("expected error loading from closed store")
	}

	if event.Classify(err) != event.Infrastructure {
		t.Errorf("expected Infrastructure classification, got %s: %v", event.Classify(err), err)
	}
}

func TestNewCommandStore(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewCommandStore(db)
	if err != nil {
		t.Fatalf("NewCommandStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil command store")
	}
}

func TestNewQueryStore(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	store, err := turso.NewQueryStore(db)
	if err != nil {
		t.Fatalf("NewQueryStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil query store")
	}
}

func TestNewTursoCommandStoreAlias(t *testing.T) {
	t.Parallel()

	if turso.NewTursoCommandStore == nil {
		t.Fatal("NewTursoCommandStore alias is nil")
	}
}

func TestNewTursoQueryStoreAlias(t *testing.T) {
	t.Parallel()

	if turso.NewTursoQueryStore == nil {
		t.Fatal("NewTursoQueryStore alias is nil")
	}
}

func TestConfigurePool(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Set a non-default value first to prove ConfigurePool overwrites it.
	db.SetMaxOpenConns(10)
	if got := db.Stats().MaxOpenConnections; got != 10 {
		t.Fatalf("precondition: MaxOpenConnections = %d, want 10", got)
	}

	turso.ConfigurePool(db)

	// Embedded LibSQL serializes writes — pool must be capped at 1.
	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("after ConfigurePool: MaxOpenConnections = %d, want 1", got)
	}
}
