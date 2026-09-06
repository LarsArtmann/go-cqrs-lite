package kv

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/maypok86/otter/v2"
)

const defaultCacheCapacity = 1000

// Sentinel errors for Cache construction.
var (
	ErrNilTypedStore   = errorfamily.NewRejection("kv.cache.nil_store", "kv: store must not be nil")
	ErrInvalidCacheCap = errorfamily.NewRejection(
		"kv.cache.invalid_capacity",
		"kv: capacity must be positive",
	)
)

// Cache wraps a [TypedStore] with an in-memory Otter cache (ADR-0032).
//
// Get checks the cache first; on miss it delegates to the underlying store and
// caches the result. Set and Delete are write-through: they update the store
// first, then the cache. Has checks the cache first; on miss it delegates.
// Scan always bypasses the cache.
//
// Values are copy-isolated: Get returns a deep copy (one codec round-trip via
// the TypedStore's codec) and Set caches a private copy, so mutations by one
// caller never leak into the cache or other readers. A cache hit therefore
// costs roughly one decode; consumers with immutable values and hot read paths
// that must avoid that cost should use the underlying [TypedStore] directly.
//
// CONSISTENCY MODEL: the cache assumes a SINGLE WRITER going through this
// Cache instance. Writers that bypass it (Cache.Store(), Cache.Backend(),
// another Cache over the same store, another process) leave stale entries
// behind — the cache has no cross-instance invalidation. For such topologies
// set [WithCacheTTL] to your acceptable staleness bound and use Invalidate /
// InvalidateAll after out-of-band writes. The default TTL is 0 (unbounded
// staleness), which is only sound under the single-writer assumption.
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
		return nil, errorfamily.WrapRejection(ErrInvalidCacheCap,
			"kv.cache.invalid_capacity",
			fmt.Sprintf("capacity must be positive, got %d", cfg.capacity))
	}

	otterOpts := &otter.Options[string, *T]{ //nolint:exhaustruct_v5 // only MaximumSize needed by default
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
// Negative results (ErrNotFound) are NOT cached. The returned value is a deep
// copy owned by the caller; mutating it never affects the cache.
func (cs *Cache[T, K]) Get(ctx context.Context, id K) (*T, error) {
	key := id.String()

	if val, ok := cs.cache.GetIfPresent(key); ok {
		return cs.copy(val)
	}

	val, err := cs.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	cached, err := cs.copy(val)
	if err != nil {
		return nil, err
	}

	cs.cache.Set(key, cached)

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
		return false, err
	}

	return has, nil
}

// Set writes val to the store and updates the cache (write-through).
// The cache stores a private deep copy of val, so mutating val after Set
// returns never affects subsequent reads.
func (cs *Cache[T, K]) Set(ctx context.Context, id K, val *T) error {
	err := cs.store.Set(ctx, id, val)
	if err != nil {
		return err
	}

	cached, err := cs.copy(val)
	if err != nil {
		// The store write succeeded; drop any stale cached entry so the
		// next Get reflects the store instead of the pre-Set value.
		cs.cache.Invalidate(id.String())

		return err
	}

	cs.cache.Set(id.String(), cached)

	return nil
}

// Delete removes the value from the store and invalidates the cache entry.
func (cs *Cache[T, K]) Delete(ctx context.Context, id K) error {
	err := cs.store.Delete(ctx, id)
	if err != nil {
		return err
	}

	cs.cache.Invalidate(id.String())

	return nil
}

// Scan returns all values matching the prefix. Always bypasses the cache.
func (cs *Cache[T, K]) Scan(ctx context.Context, prefix []byte) ([]*T, error) {
	results, err := cs.store.Scan(ctx, prefix)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Backend returns the underlying [Store].
func (cs *Cache[T, K]) Backend() Store { return cs.store.Backend() }

// Invalidate drops the cached entry for id, forcing the next Get to read
// through to the store. Call it after writing through a handle that bypasses
// this Cache (see the consistency model on [Cache]).
func (cs *Cache[T, K]) Invalidate(id K) {
	cs.cache.Invalidate(id.String())
}

// InvalidateAll drops every cached entry. Call it after out-of-band writes
// or projection resets touching this namespace (e.g. after TypedStore
// DeleteAll) so subsequent Gets re-read the store.
func (cs *Cache[T, K]) InvalidateAll() {
	cs.cache.InvalidateAll()
}

// copy deep-clones val via the TypedStore's codec so cached entries are never
// shared with callers. Types that cannot round-trip their codec already fail
// in TypedStore.Get, so failures here indicate the same class of breakage.
func (cs *Cache[T, K]) copy(val *T) (*T, error) {
	data, err := cs.store.codec.Encode(val)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"kv.cache.copy_encode",
			"copy value for cache isolation",
		)
	}

	var out T

	err = cs.store.codec.Decode(data, &out)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"kv.cache.copy_decode",
			"copy value for cache isolation",
		)
	}

	return &out, nil
}

// Store returns the underlying unwrapped [TypedStore].
func (cs *Cache[T, K]) Store() *TypedStore[T, K] { return cs.store }

// Close is currently a no-op. Otter v2 manages cleanup via finalizers.
// Retained for API stability — future otter versions may require explicit cleanup.
func (cs *Cache[T, K]) Close() {}
