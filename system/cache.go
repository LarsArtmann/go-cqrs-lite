package system

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/maypok86/otter/v2"
)

// CachedEventStore wraps an event.Store with a read-through cache.
//
// Write ordering: Save and AppendBatch drop the cached entry BEFORE the
// store write and again after it, and Load only repopulates the cache when
// no write touched the stream in between (the generation guard below).
// Invalidation alone — before or after the write — leaves a window where a
// concurrent Load re-populates the pre-save snapshot and serves it forever.
type CachedEventStore struct {
	store    event.Store
	cache    *otter.Cache[string, []event.Event]
	capacity int

	mu   sync.Mutex
	gens map[string]uint64
}

// NewCachedEventStore wraps an event.Store with a read-through cache.
func NewCachedEventStore(store event.Store, capacity int) (*CachedEventStore, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrCacheCapacityInvalid, capacity)
	}

	cache := otter.Must(&otter.Options[string, []event.Event]{
		MaximumSize: capacity,
	})

	return &CachedEventStore{
		store:    store,
		cache:    cache,
		capacity: capacity,
		gens:     make(map[string]uint64),
	}, nil
}

func (c *CachedEventStore) Save(
	ctx context.Context, ref id.StreamRef, events []event.Event, expectedVersion event.Version,
) error {
	key := ref.StreamKey()
	c.beginWrite(key)
	err := c.store.Save(ctx, ref, events, expectedVersion)
	c.endWrite(key)

	return err
}

func (c *CachedEventStore) AppendBatch(
	ctx context.Context, ref id.StreamRef, events []event.Event,
) error {
	key := ref.StreamKey()
	c.beginWrite(key)
	err := c.store.AppendBatch(ctx, ref, events)
	c.endWrite(key)

	return err
}

func (c *CachedEventStore) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	key := ref.StreamKey()
	if events, ok := c.cache.GetIfPresent(key); ok {
		return events, nil
	}

	gen := c.readGeneration(key)

	events, err := c.store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Only cache when no write began or completed during the store read;
	// otherwise this snapshot is (or may be) pre-save and must not survive.
	if c.generationUnchanged(key, gen) {
		c.cache.Set(key, events)
	}

	return events, nil
}

// advanceGeneration bumps the stream's write generation under the cache lock.
func (c *CachedEventStore) advanceGeneration(key string) {
	c.mu.Lock()
	c.gens[key]++
	c.mu.Unlock()
}

// beginWrite opens a write: the generation advances and the cached entry is
// dropped BEFORE the store write, so no reader is served the pre-write
// snapshot while the write is in flight.
func (c *CachedEventStore) beginWrite(key string) {
	c.advanceGeneration(key)

	c.cache.Invalidate(key)
}

// endWrite closes a write: the entry is dropped again (a concurrent Load may
// have repopulated during the store write) and the generation advances so
// in-flight Loads never cache a pre-write snapshot.
//
// Generation entries are intentionally never deleted: an absent entry must
// unambiguously mean "never written" (generation 0). Pruning would let a
// Load that captured 0 before a write see absent-0 again after the write
// and cache a stale snapshot. The map is bounded by the number of distinct
// streams ever written through this process — the same order as the store's
// own stream index.
func (c *CachedEventStore) endWrite(key string) {
	c.cache.Invalidate(key)

	c.advanceGeneration(key)
}

// readGeneration snapshots the stream's write generation before a store read.
func (c *CachedEventStore) readGeneration(key string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.gens[key]
}

// generationUnchanged reports whether no write began or completed since gen
// was captured. Absent means generation 0 (never written) — see endWrite for
// why entries are never pruned.
func (c *CachedEventStore) generationUnchanged(key string, gen uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.gens[key] == gen
}

func (c *CachedEventStore) LoadFromVersion(
	ctx context.Context, ref id.StreamRef, version event.Version,
) ([]event.Event, error) {
	return c.store.LoadFromVersion(ctx, ref, version)
}

func (c *CachedEventStore) LoadToVersion(
	ctx context.Context, ref id.StreamRef, maxVersion event.Version,
) ([]event.Event, error) {
	return c.store.LoadToVersion(ctx, ref, maxVersion)
}

func (c *CachedEventStore) LoadToTimestamp(
	ctx context.Context, ref id.StreamRef, maxTime time.Time,
) ([]event.Event, error) {
	return c.store.LoadToTimestamp(ctx, ref, maxTime)
}

func (c *CachedEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	if j, ok := c.store.(event.Journal); ok {
		return j.ReadAll(ctx)
	}

	return nil, ErrJournalMissing
}

func (c *CachedEventStore) ReadFrom(
	ctx context.Context, afterEventID id.EventID, limit int,
) ([]event.Event, error) {
	if sj, ok := c.store.(event.SeekableJournal); ok {
		return sj.ReadFrom(ctx, afterEventID, limit)
	}

	return nil, ErrSeekableJournalMissing
}

// CacheStats returns basic cache statistics for introspection.
func (c *CachedEventStore) CacheStats() (size int, capacity int) {
	return c.cache.EstimatedSize(), c.capacity
}
