package metaengine

import (
	"context"
	"fmt"
	"reflect"
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

	if err := r.store.IsPoisoned(r.collection); err != nil {
		return zero, false, err
	}

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

		return v, true, err
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

		return v, true, err
	}

	return zero, false, fmt.Errorf("%w: %s", errUnsupportedMapReads, eng.Profile().Name)
}

// Scan returns all values matching the given filter/sort/limit options.
// Uses raw scan when available for single-pass decode per row.
func (r *TypedReader[V]) Scan(ctx context.Context, opts ...ScanOption) ([]V, error) {
	if err := r.store.IsPoisoned(r.collection); err != nil {
		return nil, err
	}

	cfg := scanConfig{limit: 100}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Expand range specs into filter pairs for pushdown/raw paths.
	for _, rg := range cfg.ranges {
		cfg.filters = append(
			cfg.filters,
			FilterSpec{Column: rg.Column, Op: FilterGe, Value: rg.Low},
			FilterSpec{Column: rg.Column, Op: FilterLe, Value: rg.High},
		)
	}

	// Expand IN specs into FilterIn FilterSpecs so pushdown/raw paths
	// handle them. Without this, WithIn is silently dropped on
	// RawScanReader and PushdownScan paths (data correctness bug).
	for _, in := range cfg.inSpecs {
		cfg.filters = append(cfg.filters, FilterSpec{
			Column: in.Column,
			Op:     FilterIn,
			Value:  in.Values,
		})
	}

	// IN specs are now merged into filters — clear so closure fallback
	// doesn't double-evaluate.
	cfg.inSpecs = nil

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
		var filterFn func(item any) bool

		if len(cfg.filters) > 0 || len(cfg.inSpecs) > 0 {
			inSpecs := cfg.inSpecs
			filters := cfg.filters
			filterFn = func(item any) bool {
				if !passesFilterSpecs(item, filters) {
					return false
				}

				for _, in := range inSpecs {
					val := itemFieldByName(item, in.Column)
					found := false

					for _, v := range in.Values {
						if reflect.DeepEqual(val, v) {
							found = true

							break
						}
					}

					if !found {
						return false
					}
				}

				return true
			}
		}

		var sortFn func(a, b any) int

		if cfg.sort != nil {
			col := cfg.sort.Column
			sortFn = func(a, b any) int {
				return compareValue(itemFieldByName(a, col), itemFieldByName(b, col))
			}

			if cfg.sort.Desc {
				baseSort := sortFn
				sortFn = func(a, b any) int { return -baseSort(a, b) }
			}
		}

		rows, err := sb.MapScan(ctx, r.collection, filterFn, sortFn, cfg.cursor, cfg.limit)
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
func (r *TypedReader[V]) Count(ctx context.Context, opts ...ScanOption) (int, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if ar, ok := eng.(AggregateReader); ok {
			n, err := ar.Aggregate(ctx, r.collection, AggregateCount, "", filters)
			if err != nil {
				return 0, err
			}
			return int(n), nil
		}
	}

	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return 0, err
	}

	return len(rows), nil
}

// Sum returns the sum of a numeric column across matching rows.
// Uses SQL SUM pushdown when the engine supports it.
func (r *TypedReader[V]) Sum(ctx context.Context, column string, opts ...ScanOption) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateSum, column, opts...)
}

// Min returns the minimum value of a column across matching rows.
func (r *TypedReader[V]) Min(ctx context.Context, column string, opts ...ScanOption) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateMin, column, opts...)
}

// Max returns the maximum value of a column across matching rows.
func (r *TypedReader[V]) Max(ctx context.Context, column string, opts ...ScanOption) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateMax, column, opts...)
}

// Avg returns the average value of a column across matching rows.
func (r *TypedReader[V]) Avg(ctx context.Context, column string, opts ...ScanOption) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateAvg, column, opts...)
}

// aggregatePushdown tries SQL pushdown for a single aggregate function.
// Falls back to in-Go computation via Scan when the engine doesn't support
// AggregateReader.
func (r *TypedReader[V]) aggregatePushdown(
	ctx context.Context,
	fn AggregateFn,
	column string,
	opts ...ScanOption,
) (float64, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if ar, ok := eng.(AggregateReader); ok {
			return ar.Aggregate(ctx, r.collection, fn, column, filters)
		}
	}

	// Fallback: in-Go aggregation via Scan.
	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return 0, err
	}

	var result float64

	for _, row := range rows {
		val := extractValueByName(row, column)
		if n, ok := toFloat64(val); ok {
			switch fn {
			case AggregateSum, AggregateAvg:
				result += n
			case AggregateMin:
				if result == 0 || n < result {
					result = n
				}
			case AggregateMax:
				if n > result {
					result = n
				}
			}
		}
	}

	if fn == AggregateAvg && len(rows) > 0 {
		return result / float64(len(rows)), nil
	}

	return result, nil
}

// buildScanFilters applies scan options and returns the expanded filter list
// (ranges and IN specs expanded into FilterSpecs).
func buildScanFilters(opts ...ScanOption) []FilterSpec {
	cfg := scanConfig{limit: 100}
	for _, opt := range opts {
		opt(&cfg)
	}

	for _, rg := range cfg.ranges {
		cfg.filters = append(cfg.filters,
			FilterSpec{Column: rg.Column, Op: FilterGe, Value: rg.Low},
			FilterSpec{Column: rg.Column, Op: FilterLe, Value: rg.High},
		)
	}

	for _, in := range cfg.inSpecs {
		cfg.filters = append(cfg.filters, FilterSpec{
			Column: in.Column, Op: FilterIn, Value: in.Values,
		})
	}

	return cfg.filters
}

// Distinct returns the unique values of a column across matching rows.
func (r *TypedReader[V]) Distinct(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) ([]any, error) {
	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, err
	}

	seen := make(map[any]struct{}, len(rows))

	var result []any

	for _, row := range rows {
		val := extractValueByName(row, column)
		if val == nil {
			continue
		}

		if _, exists := seen[val]; exists {
			continue
		}

		seen[val] = struct{}{}
		result = append(result, val)
	}

	return result, nil
}

// --- Scan options ---

type scanConfig struct {
	filters []FilterSpec
	sort    *SortSpec
	cursor  any
	limit   int
	ranges  []RangeSpec
	inSpecs []InSpec
}

// RangeSpec declares a range filter (SQL BETWEEN) on a column.
type RangeSpec struct {
	Column string
	Low    any
	High   any
}

// InSpec declares an IN filter (SQL WHERE col IN (...)) on a column.
type InSpec struct {
	Column string
	Values []any
}

// ScanOption tunes a TypedReader.Scan call.
type ScanOption func(*scanConfig)

// WithFilter adds a comparison filter on a column.
func WithFilter(column string, op FilterOp, value any) ScanOption {
	return func(c *scanConfig) {
		c.filters = append(c.filters, FilterSpec{Column: column, Op: op, Value: value})
	}
}

// WithRange adds a range filter (low <= column <= high) on a column.
// On pushdown engines this generates SQL BETWEEN; on closure-based engines it
// generates two comparison predicates.
func WithRange(column string, low, high any) ScanOption {
	return func(c *scanConfig) {
		c.ranges = append(c.ranges, RangeSpec{Column: column, Low: low, High: high})
	}
}

// WithIn adds an IN filter (column IN values) on a column.
// On pushdown engines this generates SQL WHERE col IN (...); on closure-based
// engines it generates a membership predicate.
func WithIn(column string, values []any) ScanOption {
	return func(c *scanConfig) {
		c.inSpecs = append(c.inSpecs, InSpec{Column: column, Values: values})
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
