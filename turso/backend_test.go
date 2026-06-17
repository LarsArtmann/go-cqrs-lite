package turso_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
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

// TestBackend_FullLifecycle verifies the Backend facade end-to-end: all four
// store types (Event, Snapshot, Checkpoint, Command) coexist on a single shared
// *sql.DB with the full schema initialized. This catches cross-store schema
// conflicts and lifecycle wiring that per-store tests miss (see status report E4).
//
//nolint:tparallel // subtests share one SQLite DB; concurrent writes hit stale-snapshot errors
func TestBackend_FullLifecycle(t *testing.T) {
	t.Parallel()

	backend, cleanup := newBackendDB(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Issue", aggID)

	t.Run("EventStore_SaveLoadMultiVersion", func(t *testing.T) {
		verifyEventStoreRoundtrip(t, ctx, backend, aggID, ref)
	})

	t.Run("SnapshotStore_SaveLoadDelete", func(t *testing.T) {
		verifySnapshotStoreRoundtrip(t, ctx, backend, aggID, ref)
	})

	t.Run("CheckpointStore_SaveLoadUpdate", func(t *testing.T) {
		verifyCheckpointStoreRoundtrip(t, ctx, backend)
	})

	t.Run("CommandStore_PersistsAlongsideEvents", func(t *testing.T) {
		verifyCommandStoreRoundtrip(t, ctx, backend, aggID)
	})
}

func verifyEventStoreRoundtrip(
	t *testing.T,
	ctx context.Context,
	backend *storage.SQLBackend,
	aggID id.AggregateID,
	ref event.AggregateRef,
) {
	t.Helper()

	store := backend.EventStore()
	if store == nil {
		t.Fatal("EventStore() returned nil")
	}

	var expectedVersion event.Version

	for i := 1; i <= 3; i++ {
		evt, err := event.NewEvent(
			"issue.updated", aggID, "Issue", event.Version(i),
			[]byte(`{"n":`+strconv.Itoa(i)+`}`),
		)
		if err != nil {
			t.Fatalf("NewEvent #%d: %v", i, err)
		}

		if err := store.Save(ctx, ref, []event.Event{evt}, expectedVersion); err != nil {
			t.Fatalf("Save #%d (expectedVersion=%d): %v", i, expectedVersion, err)
		}

		expectedVersion = event.Version(i)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 events, got %d", len(loaded))
	}

	for i, evt := range loaded {
		if evt.Version() != event.Version(i+1) {
			t.Fatalf("event[%d].Version = %d, want %d", i, evt.Version(), i+1)
		}
	}
}

func verifySnapshotStoreRoundtrip(
	t *testing.T,
	ctx context.Context,
	backend *storage.SQLBackend,
	aggID id.AggregateID,
	ref event.AggregateRef,
) {
	t.Helper()

	snapStore, err := backend.SnapshotStore()
	if err != nil {
		t.Fatalf("SnapshotStore: %v", err)
	}

	state := []byte(`{"title":"snapshot-issue","status":"open"}`)
	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       3,
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	loaded, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load snapshot: %v", err)
	}

	if loaded == nil {
		t.Fatal("Load returned nil snapshot")
	}

	if loaded.Version != 3 {
		t.Fatalf("snapshot Version = %d, want 3", loaded.Version)
	}

	if string(loaded.State) != string(state) {
		t.Fatalf("snapshot State = %q, want %q", loaded.State, state)
	}

	atVersion, err := snapStore.LoadAtVersion(ctx, ref, 3)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if atVersion.Version != 3 {
		t.Fatalf("LoadAtVersion Version = %d, want 3", atVersion.Version)
	}

	if err := snapStore.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete snapshot: %v", err)
	}

	// After Delete the snapshot is gone: Load returns nil + not-found error.
	if _, err := snapStore.Load(ctx, ref); err == nil {
		t.Fatal("Load after Delete: expected error, got snapshot")
	}
}

func verifyCheckpointStoreRoundtrip(
	t *testing.T,
	ctx context.Context,
	backend *storage.SQLBackend,
) {
	t.Helper()

	cpStore, err := backend.CheckpointStore()
	if err != nil {
		t.Fatalf("CheckpointStore: %v", err)
	}

	const projection = "issue_projection"

	// Empty projection returns zero checkpoint, no error.
	initial, err := cpStore.Load(ctx, projection)
	if err != nil {
		t.Fatalf("Load (initial): %v", err)
	}

	if !initial.IsZero() {
		t.Fatalf("initial checkpoint should be zero, got %v", initial)
	}

	firstEventID := id.NewEventID()
	if err := cpStore.Save(ctx, projection, event.Checkpoint{EventID: firstEventID}); err != nil {
		t.Fatalf("Save (first): %v", err)
	}

	loaded, err := cpStore.Load(ctx, projection)
	if err != nil {
		t.Fatalf("Load (after first save): %v", err)
	}

	if loaded.EventID != firstEventID {
		t.Fatalf("checkpoint EventID = %v, want %v", loaded.EventID, firstEventID)
	}

	// Overwrite (update) the checkpoint.
	secondEventID := id.NewEventID()
	if err := cpStore.Save(ctx, projection, event.Checkpoint{EventID: secondEventID}); err != nil {
		t.Fatalf("Save (overwrite): %v", err)
	}

	updated, err := cpStore.Load(ctx, projection)
	if err != nil {
		t.Fatalf("Load (after overwrite): %v", err)
	}

	if updated.EventID != secondEventID {
		t.Fatalf(
			"checkpoint EventID = %v, want %v (after overwrite)",
			updated.EventID,
			secondEventID,
		)
	}
}

func verifyCommandStoreRoundtrip(
	t *testing.T,
	ctx context.Context,
	backend *storage.SQLBackend,
	aggID id.AggregateID,
) {
	t.Helper()

	cmdStore, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore: %v", err)
	}

	cmdRef := command.NewAggregateRef("Issue", aggID)
	cmd, err := command.NewPersistedCommand("issue.update", cmdRef, []byte(`{"status":"open"}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := cmdStore.Save(ctx, cmdRef, cmd); err != nil {
		t.Fatalf("Save command: %v", err)
	}

	loaded, err := cmdStore.Load(ctx, cmdRef)
	if err != nil {
		t.Fatalf("Load command: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
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
