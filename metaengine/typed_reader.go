package metaengine

import (
	"context"
	"fmt"
)

// TypedReader provides typed read access to a collection's values without
// constructing a query input struct. It bridges the gap between the reflective
// Execute path and fully typed access:
//
//	reader := metaengine.NewReader[FindUserResult](store, "find_user")
//	user, found, err := reader.Get(ctx, userID)
//	open, err := reader.Scan(ctx,
//	    metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
//	    metaengine.WithSort("Priority", true),
//	    metaengine.WithLimit(10),
//	)
type TypedReader[V any] struct {
	store      *Store
	collection string
	prefetch   *PrefetchCache
}

// NewReader creates a typed reader for a collection's values. The collection
// name must match a query name registered via Plan().
func NewReader[V any](store *Store, collection string) *TypedReader[V] {
	return &TypedReader[V]{store: store, collection: collection}
}

// WithPrefetch attaches a PrefetchCache to the reader for cursor-based
// pagination caching. When set, Scan results beyond the requested limit are
// cached for the next page request, eliminating redundant engine round-trips.
func (r *TypedReader[V]) WithPrefetch(cache *PrefetchCache) *TypedReader[V] {
	r.prefetch = cache

	return r
}

// readResult packs a point-lookup result for coalescer transport.
type readResult struct {
	value any
	found bool
}

// Get performs a point lookup by key, decoding the value directly to V.
// Returns (zero, false, nil) when the key is not found.
// When a ReadCoalescer is configured on the Store, concurrent Get calls for
// the same key are coalesced into a single engine read.
func (r *TypedReader[V]) Get(ctx context.Context, key any) (V, bool, error) {
	var zero V

	if r.store.coalescer != nil {
		coalesceKey := r.collection + ":" + fmt.Sprint(key)

		result, err := r.store.coalescer.Do(coalesceKey, func() (any, error) {
			return r.getUncached(ctx, key)
		})
		if err != nil {
			return zero, false, err
		}

		rr, ok := result.(readResult)
		if !ok {
			return zero, false, fmt.Errorf("%w: %T", errCoalescerTypeMismatch, result)
		}

		if !rr.found {
			return zero, false, nil
		}

		v, err := reify[V](rr.value)

		return v, true, err
	}

	rr, err := r.getUncached(ctx, key)
	if err != nil {
		return zero, false, err
	}

	if !rr.found {
		return zero, false, nil
	}

	v, err := reify[V](rr.value)

	return v, true, err
}

// getUncached performs the actual engine read without coalescer wrapping.
func (r *TypedReader[V]) getUncached(ctx context.Context, key any) (readResult, error) {
	if err := r.store.IsPoisoned(r.collection); err != nil {
		return readResult{}, err
	}

	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return readResult{}, fmt.Errorf("%w: %q", errNoQueryForInputType, r.collection)
	}

	// Prefer raw value reader for single-pass decode (1 JSON op instead of 3).
	if rvr, ok := eng.(RawValueReader); ok {
		raw, found, err := rvr.GetRawValue(ctx, r.collection, key)
		if err != nil {
			return readResult{}, fmt.Errorf("typed reader get %s: %w", r.collection, err)
		}

		return readResult{value: jsonValue(raw), found: found}, nil
	}

	// Standard MapGet path.
	if mb, ok := eng.(MapBackend); ok {
		val, found, err := mb.MapGet(ctx, r.collection, key)
		if err != nil {
			return readResult{}, fmt.Errorf("typed reader get %s: %w", r.collection, err)
		}

		return readResult{value: val, found: found}, nil
	}

	return readResult{}, fmt.Errorf("%w: %s", errUnsupportedMapReads, eng.Profile().Name)
}

// Scan returns all values matching the given filter/sort/limit options.
// Uses raw scan when available for single-pass decode per row.
func (r *TypedReader[V]) Exists(ctx context.Context, key any) (bool, error) {
	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return false, fmt.Errorf("%w: %q", errNoQueryForInputType, r.collection)
	}

	if sb, ok := eng.(SetBackend); ok {
		return sb.SetContains(ctx, r.collection, key) //nolint:wrapcheck
	}

	if mb, ok := eng.(MapBackend); ok {
		_, found, err := mb.MapGet(ctx, r.collection, key)

		return found, err //nolint:wrapcheck // passthrough
	}

	return false, fmt.Errorf("%w: %s", errUnsupportedSetReads, eng.Profile().Name)
}

// GetBatch performs point lookups for multiple keys. Returns values in the same
// order as the input keys; missing keys are skipped from the result.
func (r *TypedReader[V]) GetBatch(ctx context.Context, keys []any) ([]V, error) {
	result := make([]V, 0, len(keys))

	for _, key := range keys {
		v, found, err := r.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		if found {
			result = append(result, v)
		}
	}

	return result, nil
}

// Count returns the number of values matching the given filter options.
// When the engine implements AggregateReader, COUNT(*) is pushed to SQL;
// otherwise it falls back to Scan + len(rows).
