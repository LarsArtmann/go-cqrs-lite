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
	ref := id.NewStreamRef("Counter", id.NewStreamID())

	_, _, ok := cache.Get(ref)
	if ok {
		t.Error("expected miss on empty cache")
	}
}

func TestStateCache_PutGet(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref := id.NewStreamRef("Counter", id.NewStreamID())

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
	ref := id.NewStreamRef("Counter", id.NewStreamID())

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
	ref := id.NewStreamRef("Counter", id.NewStreamID())

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

func TestStateCache_CapacityBounded(t *testing.T) {
	t.Parallel()

	// Verify the cache accepts a capacity and doesn't panic on overflow.
	// Otter's TinyLFU evicts lazily during maintenance — we don't assert
	// exact eviction count here (that's otter's responsibility, tested upstream).
	cache := decider.NewStateCache[counterState](10)

	for i := range 50 {
		ref := id.NewStreamRef("Counter", id.NewStreamID())
		cache.Put(ref, counterState{Value: i}, event.Version(i))
	}

	// Cache should not panic and should remain usable.
	ref := id.NewStreamRef("Counter", id.NewStreamID())
	cache.Put(ref, counterState{Value: 99}, event.Version(99))

	state, _, ok := cache.Get(ref)
	if !ok || state.Value != 99 {
		t.Errorf("expected last entry to be retrievable, got ok=%v value=%d", ok, state.Value)
	}
}

func TestStateCache_FrequencyProtectsHotEntry(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)

	hotRef := id.NewStreamRef("Counter", id.NewStreamID())
	cache.Put(hotRef, counterState{Value: 1}, event.Version(1))

	// Access hotRef many times to build frequency in the TinyLFU sketch.
	for range 20 {
		_, _, _ = cache.Get(hotRef)
	}

	// Fill the cache with cold entries.
	for i := range 10 {
		coldRef := id.NewStreamRef("Counter", id.NewStreamID())
		cache.Put(coldRef, counterState{Value: i}, event.Version(i))
	}

	// The frequently-accessed entry should survive eviction.
	if _, _, ok := cache.Get(hotRef); !ok {
		t.Error("expected hot entry to survive due to high access frequency")
	}
}

func TestStateCache_DefaultCapacity(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](0)
	if cache == nil {
		t.Fatal("expected non-nil cache with capacity=0")
	}

	// Should work with default capacity without panicking
	ref := id.NewStreamRef("Counter", id.NewStreamID())
	cache.Put(ref, counterState{Value: 1}, event.Version(1))

	state, _, ok := cache.Get(ref)
	if !ok || state.Value != 1 {
		t.Errorf("expected hit with Value=1, got ok=%v Value=%d", ok, state.Value)
	}
}

func TestStateCache_Concurrent(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](100)
	ref := id.NewStreamRef("Counter", id.NewStreamID())

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

func TestStateCache_PerStreamIsolation(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[counterState](10)
	ref1 := id.NewStreamRef("Counter", id.NewStreamID())
	ref2 := id.NewStreamRef("Counter", id.NewStreamID())

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
	ref := id.NewStreamRef("Counter", id.NewStreamID())

	// Should not panic
	cache.Invalidate(ref)
}
