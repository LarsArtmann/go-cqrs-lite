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
}

// NewReader creates a typed reader for a collection's values. The collection
// name must match a query name registered via Plan().
func NewReader[V any](store *Store, collection string) *TypedReader[V] {
	return &TypedReader[V]{store: store, collection: collection}
}

// Get performs a point lookup by key, decoding the value directly to V.
// Returns (zero, false, nil) when the key is not found.
func (r *TypedReader[V]) Get(ctx context.Context, key any) (V, bool, error) {
	var zero V

	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return zero, false, fmt.Errorf("%w: %q", errNoQueryForInputType, r.collection)
	}

	// Prefer raw value reader for single-pass decode (1 JSON op instead of 3).
	if rvr, ok := eng.(RawValueReader); ok {
		raw, found, err := rvr.GetRawValue(ctx, r.collection, key)
		if err != nil {
			return zero, false, fmt.Errorf("typed reader get %s: %w", r.collection, err)
		}

		if !found {
			return zero, false, nil
		}

		v, err := reify[V](jsonValue(raw))

		return v, true, err //nolint:wrapcheck // reify already wraps
	}

	// Standard MapGet path.
	if mb, ok := eng.(MapBackend); ok {
		val, found, err := mb.MapGet(ctx, r.collection, key)
		if err != nil {
			return zero, false, fmt.Errorf("typed reader get %s: %w", r.collection, err)
		}

		if !found {
			return zero, false, nil
		}

		v, err := reify[V](val)

		return v, true, err //nolint:wrapcheck // reify already wraps
	}

	return zero, false, fmt.Errorf("%w: %s", errUnsupportedMapReads, eng.Profile().Name)
}

// Scan returns all values matching the given filter/sort/limit options.
// Uses raw scan when available for single-pass decode per row.
func (r *TypedReader[V]) Scan(ctx context.Context, opts ...ScanOption) ([]V, error) {
	cfg := scanConfig{limit: 100}

	for _, opt := range opts {
		opt(&cfg)
	}

	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return nil, fmt.Errorf("%w: %q", errNoQueryForInputType, r.collection)
	}

	// Fastest path: raw scan → direct decode per row (1 JSON op instead of 3).
	if rsr, ok := eng.(RawScanReader); ok {
		rawRows, err := rsr.ScanRawValues(
			ctx, r.collection, cfg.filters, cfg.sort, cfg.cursor, cfg.limit,
		)
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result := make([]V, 0, len(rawRows))

		for _, raw := range rawRows {
			v, err := reify[V](jsonValue(raw))
			if err != nil {
				return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
			}

			result = append(result, v)
		}

		return trimToLimit(result, cfg.limit), nil
	}

	// Standard pushdown scan (decoded values).
	if pushdown, ok := eng.(PushdownScan); ok {
		rows, err := pushdown.PushdownMapScan(
			ctx, r.collection, cfg.filters, cfg.sort, cfg.cursor, cfg.limit,
		)
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result := make([]V, 0, len(rows))

		for _, row := range rows {
			v, err := reify[V](row)
			if err != nil {
				return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
			}

			result = append(result, v)
		}

		return trimToLimit(result, cfg.limit), nil
	}

	// Closure-based fallback (in-Go filter + sort).
	if sb, ok := eng.(ScanBackend); ok {
		rows, err := sb.MapScan(ctx, r.collection, nil, nil, cfg.cursor, cfg.limit)
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result := make([]V, 0, len(rows))

		for _, row := range rows {
			v, err := reify[V](row)
			if err != nil {
				return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
			}

			result = append(result, v)
		}

		return trimToLimit(result, cfg.limit), nil
	}

	return nil, fmt.Errorf("%w: %s", errUnsupportedScanReads, eng.Profile().Name)
}

// Exists checks whether a key is present in the collection.
// Uses SetBackend.SetContains for Set ADTs, falls back to MapGet for Map ADTs.
func (r *TypedReader[V]) Exists(ctx context.Context, key any) (bool, error) {
	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return false, fmt.Errorf("%w: %q", errNoQueryForInputType, r.collection)
	}

	if sb, ok := eng.(SetBackend); ok {
		return sb.SetContains(ctx, r.collection, key)
	}

	if mb, ok := eng.(MapBackend); ok {
		_, found, err := mb.MapGet(ctx, r.collection, key)

		return found, err //nolint:wrapcheck // passthrough
	}

	return false, fmt.Errorf("%w: %s", errUnsupportedSetReads, eng.Profile().Name)
}

// --- Scan options ---

type scanConfig struct {
	filters []FilterSpec
	sort    *SortSpec
	cursor  any
	limit   int
}

// ScanOption tunes a TypedReader.Scan call.
type ScanOption func(*scanConfig)

// WithFilter adds an equality filter on a column.
func WithFilter(column string, op FilterOp, value any) ScanOption {
	return func(c *scanConfig) {
		c.filters = append(c.filters, FilterSpec{Column: column, Op: op, Value: value})
	}
}

// WithSort sets the sort column and direction.
func WithSort(column string, desc bool) ScanOption {
	return func(c *scanConfig) {
		c.sort = &SortSpec{Column: column, Desc: desc}
	}
}

// WithLimit sets the maximum number of results.
func WithLimit(n int) ScanOption {
	return func(c *scanConfig) { c.limit = n }
}

// WithCursor sets the keyset pagination cursor (the last sort key from the
// previous page).
func WithCursor(v any) ScanOption {
	return func(c *scanConfig) { c.cursor = v }
}

// trimToLimit trims the result to the limit (engines return limit+1 rows for
// has-more detection, which TypedReader.Scan discards).
func trimToLimit[V any](result []V, limit int) []V {
	if limit > 0 && len(result) > limit {
		return result[:limit]
	}

	return result
}
