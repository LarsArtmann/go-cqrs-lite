package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// CachedEventStore wraps an event.Store with a read-through cache.
type CachedEventStore struct {
	store    event.Store
	cache    *otter.Cache[string, []event.Event]
	capacity int
}

// NewCachedEventStore wraps an event.Store with a read-through cache.
func NewCachedEventStore(store event.Store, capacity int) (*CachedEventStore, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrCacheCapacityInvalid, capacity)
	}

	cache := otter.Must(&otter.Options[string, []event.Event]{
		MaximumSize: capacity,
	})

	return &CachedEventStore{store: store, cache: cache, capacity: capacity}, nil
}

func (c *CachedEventStore) Save(
	ctx context.Context, ref id.StreamRef, events []event.Event, expectedVersion event.Version,
) error {
	return c.store.Save(ctx, ref, events, expectedVersion)
}

func (c *CachedEventStore) AppendBatch(
	ctx context.Context, ref id.StreamRef, events []event.Event,
) error {
	return c.store.AppendBatch(ctx, ref, events)
}

func (c *CachedEventStore) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	key := ref.StreamKey()
	if events, ok := c.cache.GetIfPresent(key); ok {
		return events, nil
	}

	events, err := c.store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	c.cache.Set(key, events)

	return events, nil
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
