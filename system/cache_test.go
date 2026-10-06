package system_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// slowLoadStore delays every Load so a reader's store round-trip can
// straddle a concurrent Save commit — the window the idea-233 race lives in.
type slowLoadStore struct {
	event.Store
	delay time.Duration
}

func (s *slowLoadStore) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	time.Sleep(s.delay)

	return s.Store.Load(ctx, ref)
}

func newCacheTestEvent(t *testing.T, ref id.StreamRef, version event.Version) event.Event {
	t.Helper()

	return eventtest.NewEvent(
		t,
		"cache.test",
		ref.ID,
		ref.Type,
		version,
		[]byte(`{"kind":"cache"}`),
	)
}

// TestCachedEventStore_ConcurrentLoadSaveNeverServesPreSaveSnapshot is the
// race regression for the stale-read window: a Load whose store round-trip
// straddles a Save commit must never re-populate (and later serve) the
// pre-save snapshot. Writers assert the post-Save read contract directly;
// racing readers hammer Load to trigger the interleaving.
func TestCachedEventStore_ConcurrentLoadSaveNeverServesPreSaveSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheRace", id.NewStreamID())
	store := &slowLoadStore{Store: eventtest.NewFakeStore(), delay: 2 * time.Millisecond}

	cached, err := system.NewCachedEventStore(store, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	stop := make(chan struct{})

	readerWG := sync.WaitGroup{}

	readerErr := make(chan error, 8)

	for range 4 {
		readerWG.Add(1)

		go func() {
			defer readerWG.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				if _, err := cached.Load(ctx, ref); err != nil {
					select {
					case readerErr <- err:
					default:
					}

					return
				}
			}
		}()
	}

	const saves = 100

	writerErr := make(chan error, 1)

	go func() {
		defer func() { close(stop) }()

		for i := range saves {
			events := []event.Event{newCacheTestEvent(t, ref, event.Version(i+1))}
			if err := cached.Save(ctx, ref, events, event.Version(i)); err != nil {
				writerErr <- err

				return
			}

			loaded, err := cached.Load(ctx, ref)
			if err != nil {
				writerErr <- err

				return
			}

			if len(loaded) != i+1 {
				writerErr <- fmt.Errorf("served pre-save snapshot: got %d events, want %d", len(loaded), i+1)

				return
			}
		}

		writerErr <- nil
	}()

	readerWG.Wait()

	if err := <-writerErr; err != nil {
		t.Fatalf("writer: %v", err)
	}

	select {
	case err := <-readerErr:
		t.Fatalf("reader: %v", err)
	default:
	}
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
		t.Fatalf(
			"post-write Load: want 2 events (cache invalidated), got %d — cache is stale",
			len(loaded),
		)
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

	if err := cached.AppendBatch(
		ctx,
		ref,
		[]event.Event{newCacheTestEvent(t, ref, 2)},
	); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err = cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-append Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf(
			"post-append Load: want 2 events (cache invalidated), got %d — cache is stale",
			len(loaded),
		)
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
	if err := cached.Save(
		ctx,
		ref,
		[]event.Event{newCacheTestEvent(t, ref, 2)},
		1,
	); !errors.Is(
		err,
		boom,
	) {
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
