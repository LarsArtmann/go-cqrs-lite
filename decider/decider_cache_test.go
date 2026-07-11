package decider_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func newCacheRepo(
	t *testing.T,
	cache decider.StateCache[counterState],
	opts ...decider.RepositoryOption[counterState],
) (*decider.Repository[counterState], *eventtest.FakeStore, *eventtest.FakeBus) {
	t.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	allOpts := append(
		[]decider.RepositoryOption[counterState]{
			decider.WithStateCache[counterState](cache),
		},
		opts...,
	)

	repo, err := decider.NewRepository(store, bus, counterDecider(), allOpts...)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, store, bus
}

func TestRepository_StateCache_LoadAfterExecute(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, _, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()

	executeCreate(t, repo, aggID)

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 1 {
		t.Errorf("Value = %d, want 1", state.Value)
	}

	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}

	ref := id.NewAggregateRef("Counter", aggID)
	cachedState, cachedVersion, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected cache hit after Execute + Load")
	}

	if cachedState.Value != 1 {
		t.Errorf("cached Value = %d, want 1", cachedState.Value)
	}

	if cachedVersion != 1 {
		t.Errorf("cached version = %d, want 1", cachedVersion)
	}
}

func TestRepository_StateCache_LoadSkipsFullLoadOnHit(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		decider.WithStateCache[counterState](cache),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	// Create + seed some events
	executeCreate(t, repo, aggID)
	for i := 0; i < 4; i++ {
		_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")
	}

	// Load to populate cache
	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load (warm): %v", err)
	}

	// Wrap store to count Load calls
	counting := &countLoadStore{Store: store}

	repo2, err := decider.NewRepository(
		counting, bus, counterDecider(),
		decider.WithStateCache[counterState](cache),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	before := counting.count.Load()

	// Load again — should hit cache and NOT call store.Load
	_, _, err = repo2.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load (cached): %v", err)
	}

	if after := counting.count.Load(); after > before {
		t.Errorf("expected zero store.Load calls on cache hit, got %d", after-before)
	}
}

func TestRepository_StateCache_ExecuteUpdatesCache(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, _, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()

	// Execute creates → cache should have state at version 1
	executeCreate(t, repo, aggID)

	ref := id.NewAggregateRef("Counter", aggID)
	state, version, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected cache populated after Execute")
	}

	if state.Value != 1 {
		t.Errorf("cached Value = %d, want 1", state.Value)
	}

	if version != 1 {
		t.Errorf("cached version = %d, want 1", version)
	}

	// Execute increment → cache should be updated to version 2
	_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")

	state, version, ok = cache.Get(ref)
	if !ok {
		t.Fatal("expected cache still populated after Execute")
	}

	if state.Value != 2 {
		t.Errorf("cached Value = %d, want 2", state.Value)
	}

	if version != 2 {
		t.Errorf("cached version = %d, want 2", version)
	}
}

func TestRepository_StateCache_ColdMissPopulates(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, store, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()

	// Seed events directly via store (bypassing repo/cache)
	evt1 := makeEvent(t, "CounterCreated", aggID, 1)
	evt2 := makeEvent(t, "CounterIncremented", aggID, 2)
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt1, evt2})

	// Load — cache is cold, should populate from store
	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 2 {
		t.Errorf("Value = %d, want 2", state.Value)
	}

	if version != 2 {
		t.Errorf("version = %d, want 2", version)
	}

	// Verify cache was populated
	ref := id.NewAggregateRef("Counter", aggID)
	cachedState, cachedVersion, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected cache populated after cold Load")
	}

	if cachedState.Value != 2 || cachedVersion != 2 {
		t.Errorf("cache = {%d, v%d}, want {2, v2}", cachedState.Value, cachedVersion)
	}
}

func TestRepository_StateCache_LoadFromVersionError(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, store, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	// Pre-populate cache with stale state
	cache.Put(ref, counterState{Value: 99}, event.Version(5))

	// Make LoadFromVersion error
	store.LoadFromVersionFn(func(_ id.AggregateRef, _ event.Version) ([]event.Event, error) {
		return nil, errors.New("store unavailable")
	})

	// Should fall back to full Load and invalidate cache
	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("expected fallback Load to succeed, got: %v", err)
	}

	if state.Value != 0 {
		t.Errorf("expected initial state (no events), got Value=%d", state.Value)
	}

	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}

	// The fallback path (loadFromStore) should have run and repopulated the cache
	cachedState, _, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected cache repopulated after fallback")
	}

	if cachedState.Value != 0 {
		t.Errorf("cached Value = %d, want 0", cachedState.Value)
	}
}

func TestRepository_StateCache_FoldErrorInvalidates(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	repo, err := decider.NewRepository(
		store, bus, failingDecider(),
		decider.WithStateCache[counterState](cache),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	// Seed the store with two events (index-based: v1 at idx 0, v2 at idx 1)
	evt1 := makeEvent(t, "CounterCreated", aggID, 1)
	evt2 := makeEvent(t, "CounterIncremented", aggID, 2)
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt1, evt2})

	// Pre-populate cache with state at version 1 (bypasses the fold)
	cache.Put(ref, counterState{Value: 1}, event.Version(1))

	// Load should hit cache at v1, LoadFromVersion(v1) returns [evt2],
	// fold fails because failingDecider.Apply always errors
	_, _, err = repo.Load(context.Background(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected fold error")
	}

	if _, _, ok := cache.Get(ref); ok {
		t.Error("expected cache invalidated after fold error")
	}
}

func TestRepository_StateCache_NoEventsReturnsCached(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, _, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()

	// Create the aggregate
	executeCreate(t, repo, aggID)

	// Load to populate cache
	state1, _, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load 1: %v", err)
	}

	// Load again — no new events, should return cached state directly
	state2, _, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}

	if state2.Value != state1.Value {
		t.Errorf("state2.Value = %d, want %d", state2.Value, state1.Value)
	}
}

func TestRepository_StateCache_ConcurrentLoadExecute(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	repo, _, _ := newCacheRepo(t, cache)

	aggID := id.NewAggregateID()
	executeCreate(t, repo, aggID)

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 10 {
				_, _, _ = repo.Load(context.Background(), aggID, "Counter")
			}
		}()
	}

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 5 {
				_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")
			}
		}()
	}

	wg.Wait()

	// Final load should return a consistent state
	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("final Load: %v", err)
	}

	if state.Value < 1 {
		t.Errorf("expected Value >= 1, got %d", state.Value)
	}

	if version < 1 {
		t.Errorf("expected version >= 1, got %d", version)
	}
}

// --- ReadPressure integration tests ---

func newReadPressureRepo(
	t *testing.T,
	threshold int,
) (*decider.Repository[counterState], *eventtest.FakeStore, *eventtest.FakeSnapshotStore) {
	t.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	snapStore := eventtest.NewFakeSnapshotStore()

	rp, err := snapshot.NewReadPressure(threshold)
	if err != nil {
		t.Fatalf("NewReadPressure: %v", err)
	}

	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		decider.WithSnapshotStore[counterState](snapStore),
		decider.WithCodec[counterState](codec.JSONCodec{}),
		decider.WithSnapshotStrategy[counterState](rp),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, store, snapStore
}

func TestRepository_ReadPressure_TriggersSnapshot(t *testing.T) {
	t.Parallel()

	repo, _, snapStore := newReadPressureRepo(t, 3)
	aggID := id.NewAggregateID()

	// Create the aggregate
	executeCreate(t, repo, aggID)

	// Load 3 times to build up read pressure
	for range 3 {
		_, _, err := repo.Load(t.Context(), aggID, "Counter")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
	}

	// Next write should trigger a snapshot
	err := executeAndIncrement(t, repo, aggID, "CounterIncremented")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	snaps := snapStore.Saved()
	if len(snaps) == 0 {
		t.Fatal("expected snapshot to be saved after read pressure threshold + write")
	}

	last := snaps[len(snaps)-1]
	if last.Version != 2 {
		t.Errorf("snapshot version = %d, want 2", last.Version)
	}
}

func TestRepository_ReadPressure_NoTriggerBelowThreshold(t *testing.T) {
	t.Parallel()

	repo, _, snapStore := newReadPressureRepo(t, 10)
	aggID := id.NewAggregateID()

	// Create the aggregate
	executeCreate(t, repo, aggID)

	// Load only 2 times (below threshold of 10)
	for range 2 {
		_, _, err := repo.Load(t.Context(), aggID, "Counter")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
	}

	// Write should NOT trigger a snapshot
	err := executeAndIncrement(t, repo, aggID, "CounterIncremented")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(snapStore.Saved()) > 0 {
		t.Errorf("expected no snapshot, got %d", len(snapStore.Saved()))
	}
}

func TestRepository_ReadPressure_ResetAfterSnapshot(t *testing.T) {
	t.Parallel()

	repo, _, snapStore := newReadPressureRepo(t, 2)
	aggID := id.NewAggregateID()

	// Create + build up reads + trigger snapshot
	executeCreate(t, repo, aggID)

	for range 2 {
		_, _, _ = repo.Load(t.Context(), aggID, "Counter")
	}

	_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")

	if len(snapStore.Saved()) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapStore.Saved()))
	}

	// Another write — should NOT trigger (reads were reset)
	_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")

	if len(snapStore.Saved()) != 1 {
		t.Errorf("expected still 1 snapshot after reset, got %d", len(snapStore.Saved()))
	}
}
