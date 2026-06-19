package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
)

const defaultCapacity = 1000

// Sentinel errors for CachedStore construction.
var (
	ErrNilStore        = errors.New("cache: store must not be nil")
	ErrInvalidCapacity = errors.New("cache: capacity must be positive")
)

// CachedStore wraps a [readmodel.Store] with an in-memory Otter cache.
//
// Get checks the cache first; on miss it delegates to the underlying store and
// caches the result. Set and Delete are write-through: they update the store
// first, then the cache. Has checks the cache first; on miss it delegates.
// Scan always bypasses the cache.
//
// The cache is safe for concurrent use (Otter is lock-free for reads).
type CachedStore[T any, K fmt.Stringer] struct {
	store *readmodel.Store[T, K]
	cache *otter.Cache[string, *T]
}

// Option configures a CachedStore.
type Option[T any, K fmt.Stringer] func(*config[T, K])

type config[T any, K fmt.Stringer] struct {
	capacity int
	ttl      time.Duration
}

// WithCapacity sets the maximum number of entries the cache will hold before
// eviction (TinyLFU admission policy). Default: 1000.
func WithCapacity[T any, K fmt.Stringer](n int) Option[T, K] {
	return func(c *config[T, K]) { c.capacity = n }
}

// WithTTL sets the time-to-live for cache entries after write.
// Entries expire and are evicted lazily on next access or by background
// maintenance. Default: no expiration (entries live until evicted by capacity).
func WithTTL[T any, K fmt.Stringer](d time.Duration) Option[T, K] {
	return func(c *config[T, K]) { c.ttl = d }
}

// New creates a CachedStore wrapping the given store.
//
// The cache is configured via options. By default it holds up to 1000 entries
// with no TTL. Use [WithCapacity] and [WithTTL] to customize.
//
// Call [CachedStore.Close] when done to release cache resources.
func New[T any, K fmt.Stringer](
	store *readmodel.Store[T, K],
	opts ...Option[T, K],
) (*CachedStore[T, K], error) {
	if store == nil {
		return nil, ErrNilStore
	}

	cfg := config[T, K]{capacity: defaultCapacity, ttl: 0}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.capacity <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidCapacity, cfg.capacity)
	}

	otterOpts := &otter.Options[string, *T]{ //nolint:exhaustruct // only MaximumSize needed by default
		MaximumSize: cfg.capacity,
	}

	if cfg.ttl > 0 {
		otterOpts.ExpiryCalculator = otter.ExpiryWriting[string, *T](cfg.ttl)
	}

	cache := otter.Must(otterOpts)

	return &CachedStore[T, K]{
		store: store,
		cache: cache,
	}, nil
}

// Get returns the value for id, checking the cache first.
// On cache miss, delegates to the underlying store and caches the result.
// Negative results (ErrNotFound) are NOT cached.
func (cs *CachedStore[T, K]) Get(ctx context.Context, id K) (*T, error) {
	key := id.String()

	if val, ok := cs.cache.GetIfPresent(key); ok {
		return val, nil
	}

	val, err := cs.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cache: get %s: %w", key, err)
	}

	cs.cache.Set(key, val)

	return val, nil
}

// Has reports whether a value exists for id.
// Checks the cache first; on miss, delegates to the underlying store.
func (cs *CachedStore[T, K]) Has(ctx context.Context, id K) (bool, error) {
	if _, ok := cs.cache.GetIfPresent(id.String()); ok {
		return true, nil
	}

	has, err := cs.store.Has(ctx, id)
	if err != nil {
		return false, fmt.Errorf("cache: has %s: %w", id.String(), err)
	}

	return has, nil
}

// Set writes val to the store and updates the cache (write-through).
func (cs *CachedStore[T, K]) Set(ctx context.Context, id K, val *T) error {
	err := cs.store.Set(ctx, id, val)
	if err != nil {
		return fmt.Errorf("cache: set %s: %w", id.String(), err)
	}

	cs.cache.Set(id.String(), val)

	return nil
}

// Delete removes the value from the store and invalidates the cache entry.
func (cs *CachedStore[T, K]) Delete(ctx context.Context, id K) error {
	err := cs.store.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("cache: delete %s: %w", id.String(), err)
	}

	cs.cache.Invalidate(id.String())

	return nil
}

// Scan returns all values matching the prefix. Always bypasses the cache.
func (cs *CachedStore[T, K]) Scan(ctx context.Context, prefix []byte) ([]*T, error) {
	results, err := cs.store.Scan(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("cache: scan: %w", err)
	}

	return results, nil
}

// Backend returns the underlying readmodel.Backend.
func (cs *CachedStore[T, K]) Backend() readmodel.Backend { return cs.store.Backend() }

// Store returns the underlying unwrapped readmodel.Store.
func (cs *CachedStore[T, K]) Store() *readmodel.Store[T, K] { return cs.store }

// Close is currently a no-op. Otter v2 manages cleanup via finalizers.
// Retained for API stability — future otter versions may require explicit cleanup.
func (cs *CachedStore[T, K]) Close() {}
