package kv

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"
)

const defaultCacheCapacity = 1000

// Sentinel errors for Cache construction.
var (
	ErrNilTypedStore   = errors.New("kv: store must not be nil")
	ErrInvalidCacheCap = errors.New("kv: capacity must be positive")
)

// Cache wraps a [TypedStore] with an in-memory Otter cache (ADR-0032).
//
// Get checks the cache first; on miss it delegates to the underlying store and
// caches the result. Set and Delete are write-through: they update the store
// first, then the cache. Has checks the cache first; on miss it delegates.
// Scan always bypasses the cache.
//
// The cache is safe for concurrent use (Otter is lock-free for reads).
type Cache[T any, K fmt.Stringer] struct {
	store *TypedStore[T, K]
	cache *otter.Cache[string, *T]
}

// CacheOption configures a [Cache].
type CacheOption[T any, K fmt.Stringer] func(*cacheConfig)

type cacheConfig struct {
	capacity int
	ttl      time.Duration
}

// WithCacheCapacity sets the maximum number of entries the cache will hold before
// eviction (TinyLFU admission policy). Default: 1000.
func WithCacheCapacity[T any, K fmt.Stringer](n int) CacheOption[T, K] {
	return func(c *cacheConfig) { c.capacity = n }
}

// WithCacheTTL sets the time-to-live for cache entries after write.
// Entries expire and are evicted lazily on next access or by background
// maintenance. Default: no expiration (entries live until evicted by capacity).
func WithCacheTTL[T any, K fmt.Stringer](d time.Duration) CacheOption[T, K] {
	return func(c *cacheConfig) { c.ttl = d }
}

// NewCache creates a Cache wrapping the given TypedStore.
//
// The cache is configured via options. By default it holds up to 1000 entries
// with no TTL. Use [WithCacheCapacity] and [WithCacheTTL] to customize.
//
// Call [Cache.Close] when done to release cache resources.
func NewCache[T any, K fmt.Stringer](
	store *TypedStore[T, K],
	opts ...CacheOption[T, K],
) (*Cache[T, K], error) {
	if store == nil {
		return nil, ErrNilTypedStore
	}

	cfg := cacheConfig{capacity: defaultCacheCapacity, ttl: 0}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.capacity <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidCacheCap, cfg.capacity)
	}

	otterOpts := &otter.Options[string, *T]{ //nolint:exhaustruct // only MaximumSize needed by default
		MaximumSize: cfg.capacity,
	}

	if cfg.ttl > 0 {
		otterOpts.ExpiryCalculator = otter.ExpiryWriting[string, *T](cfg.ttl)
	}

	cache := otter.Must(otterOpts)

	return &Cache[T, K]{
		store: store,
		cache: cache,
	}, nil
}

// Get returns the value for id, checking the cache first.
// On cache miss, delegates to the underlying store and caches the result.
// Negative results (ErrNotFound) are NOT cached.
func (cs *Cache[T, K]) Get(ctx context.Context, id K) (*T, error) {
	key := id.String()

	if val, ok := cs.cache.GetIfPresent(key); ok {
		return val, nil
	}

	val, err := cs.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("kv: cache get %s: %w", key, err)
	}

	cs.cache.Set(key, val)

	return val, nil
}

// Has reports whether a value exists for id.
// Checks the cache first; on miss, delegates to the underlying store.
func (cs *Cache[T, K]) Has(ctx context.Context, id K) (bool, error) {
	if _, ok := cs.cache.GetIfPresent(id.String()); ok {
		return true, nil
	}

	has, err := cs.store.Has(ctx, id)
	if err != nil {
		return false, fmt.Errorf("kv: cache has %s: %w", id.String(), err)
	}

	return has, nil
}

// Set writes val to the store and updates the cache (write-through).
func (cs *Cache[T, K]) Set(ctx context.Context, id K, val *T) error {
	err := cs.store.Set(ctx, id, val)
	if err != nil {
		return fmt.Errorf("kv: cache set %s: %w", id.String(), err)
	}

	cs.cache.Set(id.String(), val)

	return nil
}

// Delete removes the value from the store and invalidates the cache entry.
func (cs *Cache[T, K]) Delete(ctx context.Context, id K) error {
	err := cs.store.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("kv: cache delete %s: %w", id.String(), err)
	}

	cs.cache.Invalidate(id.String())

	return nil
}

// Scan returns all values matching the prefix. Always bypasses the cache.
func (cs *Cache[T, K]) Scan(ctx context.Context, prefix []byte) ([]*T, error) {
	results, err := cs.store.Scan(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("kv: cache scan: %w", err)
	}

	return results, nil
}

// Backend returns the underlying [Store].
func (cs *Cache[T, K]) Backend() Store { return cs.store.Backend() }

// Store returns the underlying unwrapped [TypedStore].
func (cs *Cache[T, K]) Store() *TypedStore[T, K] { return cs.store }

// Close is currently a no-op. Otter v2 manages cleanup via finalizers.
// Retained for API stability — future otter versions may require explicit cleanup.
func (cs *Cache[T, K]) Close() {}
