package decider_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// pointerState is the adversarial shape for cache isolation: a POINTER state
// carrying reference-bearing fields. Before the codec-cloned Get (owner
// ruling 2026-11-08, brutal-episode7 B1), concurrent Get-and-mutate on one
// entry data-raced on the shared instance.
type pointerState struct {
	Items []string
	Tags  map[string]int
}

func TestStateCache_GetIsolatesCallers(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[*pointerState](10)
	ref := id.NewStreamRef("Order", id.NewStreamID())

	cache.Put(ref, &pointerState{
		Items: []string{"a", "b"},
		Tags:  map[string]int{"x": 1},
	}, event.Version(3))

	first, _, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected hit after Put")
	}

	first.Items[0] = "MUTATED"
	first.Tags["x"] = 999
	first.Items = append(first.Items, "grown")

	second, _, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected second hit")
	}

	if second.Items[0] != "a" || len(second.Items) != 2 {
		t.Errorf("cached Items aliased the caller's mutation: %v", second.Items)
	}

	if second.Tags["x"] != 1 {
		t.Errorf("cached Tags aliased the caller's mutation: %v", second.Tags)
	}
}

func TestStateCache_UncloneableStateFallsBackToMiss(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[uncloneableState](10)
	ref := id.NewStreamRef("Fn", id.NewStreamID())

	cache.Put(ref, uncloneableState{Fn: func() {}}, event.Version(1))

	if _, _, ok := cache.Get(ref); ok {
		t.Fatal("uncloneable state must miss, never return a shared instance")
	}

	// The entry is dropped, not retried forever.
	if _, _, ok := cache.Get(ref); ok {
		t.Fatal("uncloneable entry must be invalidated after the first miss")
	}
}

type uncloneableState struct {
	Fn func()
}

// TestStateCache_ConcurrentGetMutateNoRace pins the B1 fix: many goroutines
// Get the same entry and mutate their result. Under -race this fails on the
// pre-fix shared-instance Get (verified by the episode-7 audit).
func TestStateCache_ConcurrentGetMutateNoRace(t *testing.T) {
	t.Parallel()

	cache := decider.NewStateCache[*pointerState](4)
	ref := id.NewStreamRef("Hot", id.NewStreamID())

	cache.Put(ref, &pointerState{
		Items: make([]string, 8),
		Tags:  map[string]int{"n": 0},
	}, event.Version(1))

	var wg sync.WaitGroup

	for i := range 32 {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			state, version, ok := cache.Get(ref)
			if !ok {
				t.Error("unexpected miss under load")

				return
			}

			state.Items[0] = "writer"
			state.Tags["n"] = i
			_ = version
		}(i)
	}

	wg.Wait()
}

// TestStateCache_PutIsolatesCacheFromCaller pins the intake side of the
// clone contract (audit finding 2026-11-09): the caller keeps a reference
// to the state it hands over — LoadRef returns the very instance it Put —
// so mutating it after Put must never poison the cached entry or the
// states later callers receive.
func TestStateCache_PutIsolatesCacheFromCaller(t *testing.T) {
	cache := decider.NewStateCache[*pointerState](8)
	ref := id.NewStreamRef("Put", id.NewStreamID())

	state := &pointerState{Items: []string{"journal-true"}, Tags: map[string]int{"n": 1}}
	cache.Put(ref, state, event.Version(1))

	// Caller mutates its own instance afterwards.
	state.Items[0] = "mutated"
	state.Tags["n"] = 99

	got, version, ok := cache.Get(ref)
	if !ok {
		t.Fatal("expected hit after Put")
	}

	if got.Items[0] != "journal-true" || got.Tags["n"] != 1 {
		t.Fatalf("cache was poisoned by caller mutation: items=%v tags=%v", got.Items, got.Tags)
	}

	if version != event.Version(1) {
		t.Fatalf("version: want 1, got %d", version)
	}
}
