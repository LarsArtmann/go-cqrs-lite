package kv

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-codec"
)

// NewCacheForViewStore fronts any [ViewStore] — not only [TypedStore] — with
// the same Otter cache [NewCache] provides. This is the constructor for SQL
// view stores with real columns (e.g. storage/view.SQLViewStore), which cannot
// be passed to NewCache because they are different concrete types.
//
// Copy isolation defaults to a SHALLOW struct copy (the top-level value is
// duplicated; fields that are slices, maps, or pointers stay shared). That is
// safe for the scalar-shaped views SQL read models typically use. Views whose
// callers mutate reference-bearing fields must pass [WithCacheCopyFunc] with a
// deep copy. Has falls back to Get for stores without a Has method;
// [Cache.Backend] and [Cache.Store] return nil for non-TypedStore caches.
func NewCacheForViewStore[V any, K fmt.Stringer](
	store ViewStore[V, K],
	opts ...CacheOption[V, K],
) (*Cache[V, K], error) {
	if store == nil {
		return nil, ErrNilTypedStore
	}

	return newCache[V, K](store, shallowCopy[V], opts...)
}

// newCache assembles the Cache shared by both constructors. copyFn is the
// per-constructor isolation strategy; a config-level [WithCacheCopyFunc]
// always wins over it.
func newCache[T any, K fmt.Stringer](
	store ViewStore[T, K],
	copyFn func(*T) (*T, error),
	opts ...CacheOption[T, K],
) (*Cache[T, K], error) {
	cfg := cacheConfig[T]{capacity: defaultCacheCapacity, ttl: 0, copyFn: copyFn}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.capacity <= 0 {
		return nil, errorfamily.WrapRejection(ErrInvalidCacheCap,
			"kv.cache.invalid_capacity",
			fmt.Sprintf("capacity must be positive, got %d", cfg.capacity))
	}

	if cfg.copyFn == nil {
		return nil, errorfamily.NewRejection(
			"kv.cache.nil_copy_func",
			"kv: copy function must not be nil",
		)
	}

	otterOpts := &otter.Options[string, *T]{ //nolint:exhaustruct_v5 // only MaximumSize needed by default
		MaximumSize: cfg.capacity,
	}

	if cfg.ttl > 0 {
		otterOpts.ExpiryCalculator = otter.ExpiryWriting[string, *T](cfg.ttl)
	}

	return &Cache[T, K]{
		store:  store,
		cache:  otter.Must(otterOpts),
		copyFn: cfg.copyFn,
	}, nil
}

// shallowCopy duplicates the top-level value: assignments to the caller's
// copy never reach the cached entry, but slice/map/pointer fields are shared.
func shallowCopy[T any](val *T) (*T, error) {
	out := *val

	return &out, nil
}

// codecCopy deep-clones val via a codec round-trip so cached entries are
// never shared with callers. Types that cannot round-trip their codec already
// fail in TypedStore.Get, so failures here indicate the same class of
// breakage.
func codecCopy[T any](c codec.Codec, val *T) (*T, error) {
	data, err := c.Encode(val)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"kv.cache.copy_encode",
			"copy value for cache isolation",
		)
	}

	var out T

	err = c.Decode(data, &out)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"kv.cache.copy_decode",
			"copy value for cache isolation",
		)
	}

	return &out, nil
}
