package decider_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestStateCache_GetMiss(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	_, _, ok := cache.Get(ref)
	if ok {
		t.Error("expected miss on empty cache")
	}
}

func TestStateCache_PutGet(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	cache.Put(ref, counterState{Value: 42}, event.Version(5))

	state, version, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected hit after Put")
	}

	if state.Value != 42 {
		t.Errorf("state.Value = %d, want 42", state.Value)
	}

	if version != 5 {
		t.Errorf("version = %d, want 5", version)
	}
}

func TestStateCache_Invalidate(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	cache.Put(ref, counterState{Value: 1}, event.Version(1))
	cache.Invalidate(ref)

	_, _, ok := cache.Get(ref)
	if ok {
		t.Error("expected miss after Invalidate")
	}
}

func TestStateCache_UpdateExisting(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	cache.Put(ref, counterState{Value: 1}, event.Version(1))
	cache.Put(ref, counterState{Value: 2}, event.Version(2))

	state, version, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected hit")
	}

	if state.Value != 2 {
		t.Errorf("state.Value = %d, want 2", state.Value)
	}

	if version != 2 {
		t.Errorf("version = %d, want 2", version)
	}
}

func TestStateCache_LRUEviction(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](3)

	refs := make([]id.AggregateRef, 4)
	for i := range refs {
		refs[i] = id.NewAggregateRef("Counter", id.NewAggregateID())
		cache.Put(refs[i], counterState{Value: i}, event.Version(i))
	}

	// refs[0] should be evicted (oldest, capacity=3)
	if _, _, ok := cache.Get(refs[0]); ok {
		t.Error("expected oldest entry to be evicted")
	}

	// refs[1..3] should still be present
	for i := 1; i < 4; i++ {
		if _, _, ok := cache.Get(refs[i]); !ok {
			t.Errorf("expected entry %d to be present", i)
		}
	}
}

func TestStateCache_LRU_OrderAfterGet(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](3)

	refs := make([]id.AggregateRef, 3)
	for i := range refs {
		refs[i] = id.NewAggregateRef("Counter", id.NewAggregateID())
		cache.Put(refs[i], counterState{Value: i}, event.Version(i))
	}

	// Access refs[0] to make it most-recently-used
	_, _, _ = cache.Get(refs[0]) //nolint:dogsled // testing 3-return API

	// Insert a new entry — should evict refs[1] (now the least recently used)
	newRef := id.NewAggregateRef("Counter", id.NewAggregateID())
	cache.Put(newRef, counterState{Value: 99}, event.Version(99))

	if _, _, ok := cache.Get(refs[1]); ok {
		t.Error("expected refs[1] to be evicted after Get promoted refs[0]")
	}

	if _, _, ok := cache.Get(refs[0]); !ok {
		t.Error("expected refs[0] to survive (was recently accessed)")
	}
}

func TestStateCache_DefaultCapacity(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](0)
	if cache == nil {
		t.Fatal("expected non-nil cache with capacity=0")
	}

	// Should work with default capacity without panicking
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())
	cache.Put(ref, counterState{Value: 1}, event.Version(1))

	state, _, ok := cache.Get(ref)
	if !ok || state.Value != 1 {
		t.Errorf("expected hit with Value=1, got ok=%v Value=%d", ok, state.Value)
	}
}

func TestStateCache_Concurrent(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](100)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	var wg sync.WaitGroup

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 100 {
				cache.Put(ref, counterState{Value: i}, event.Version(i))
				_, _, _ = cache.Get(ref)
			}
		}()
	}

	wg.Wait()
}

func TestStateCache_PerAggregateIsolation(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref1 := id.NewAggregateRef("Counter", id.NewAggregateID())
	ref2 := id.NewAggregateRef("Counter", id.NewAggregateID())

	cache.Put(ref1, counterState{Value: 10}, event.Version(10))
	cache.Put(ref2, counterState{Value: 20}, event.Version(20))

	s1, _, _ := cache.Get(ref1)
	s2, _, _ := cache.Get(ref2)

	if s1.Value != 10 {
		t.Errorf("ref1 Value = %d, want 10", s1.Value)
	}

	if s2.Value != 20 {
		t.Errorf("ref2 Value = %d, want 20", s2.Value)
	}
}

func TestStateCache_InvalidateMissing(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	// Should not panic
	cache.Invalidate(ref)
}
