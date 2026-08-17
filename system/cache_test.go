package system_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func newCacheTestEvent(t *testing.T, ref id.StreamRef, version event.Version) event.Event {
	t.Helper()

	return eventtest.NewEvent(t, "cache.test", ref.ID, ref.Type, version, []byte(`{"kind":"cache"}`))
}

// TestCachedEventStore_SaveInvalidatesCache is the regression test for the
// stale-cache bug: a Save followed by a cached Load must return the freshly
// saved events, not the pre-write snapshot.
func TestCachedEventStore_SaveInvalidatesCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheTest", id.NewStreamID())
	store := eventtest.NewFakeStore()
	cached, err := system.NewCachedEventStore(store, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	first := newCacheTestEvent(t, ref, 1)
	if err := cached.Save(ctx, ref, []event.Event{first}, 0); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	loaded, err := cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("first Load: want 1 event, got %d", len(loaded))
	}

	second := newCacheTestEvent(t, ref, 2)
	if err := cached.Save(ctx, ref, []event.Event{second}, 1); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	loaded, err = cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-write Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("post-write Load: want 2 events (cache invalidated), got %d — cache is stale", len(loaded))
	}
}

// TestCachedEventStore_AppendBatchInvalidatesCache mirrors the Save test for
// the AppendBatch path.
func TestCachedEventStore_AppendBatchInvalidatesCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheTest", id.NewStreamID())
	store := eventtest.NewFakeStore()
	cached, err := system.NewCachedEventStore(store, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{newCacheTestEvent(t, ref, 1)}, 0); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	loaded, err := cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("warm Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("warm Load: want 1 event, got %d", len(loaded))
	}

	if err := cached.AppendBatch(ctx, ref, []event.Event{newCacheTestEvent(t, ref, 2)}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err = cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-append Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("post-append Load: want 2 events (cache invalidated), got %d — cache is stale", len(loaded))
	}
}

// TestCachedEventStore_CacheHitAvoidsStoreRoundTrip pins the read-through
// benefit: a second Load of the same ref must be served from the cache without
// hitting the underlying store.
func TestCachedEventStore_CacheHitAvoidsStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheTest", id.NewStreamID())
	store := eventtest.NewFakeStore()

	calls := 0
	store.LoadFn(func(ref id.StreamRef) ([]event.Event, error) {
		calls++
		if calls > 1 {
			return nil, errors.New("cache miss: store should not be hit twice")
		}

		return []event.Event{newCacheTestEvent(t, ref, 1)}, nil
	})

	cached, err := system.NewCachedEventStore(store, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	for range 2 {
		loaded, err := cached.Load(ctx, ref)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(loaded) != 1 {
			t.Fatalf("Load: want 1 event, got %d", len(loaded))
		}
	}
}

// TestCachedEventStore_SaveErrorKeepsCacheEntry ensures a failed write does
// NOT evict a still-valid cache entry (no unnecessary store round-trip).
func TestCachedEventStore_SaveErrorKeepsCacheEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheTest", id.NewStreamID())
	store := eventtest.NewFakeStore()
	cached, err := system.NewCachedEventStore(store, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{newCacheTestEvent(t, ref, 1)}, 0); err != nil {
		t.Fatalf("seed Save: %v", err)
	}
	if _, err := cached.Load(ctx, ref); err != nil { // warm the cache
		t.Fatalf("warm Load: %v", err)
	}

	boom := errors.New("boom")
	store.SaveFn(func(context.Context, id.StreamRef, []event.Event, event.Version) error {
		return boom
	})
	if err := cached.Save(ctx, ref, []event.Event{newCacheTestEvent(t, ref, 2)}, 1); !errors.Is(err, boom) {
		t.Fatalf("failed Save: want boom, got %v", err)
	}

	loaded, err := cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-error Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("post-error Load: want 1 cached event, got %d", len(loaded))
	}
}
