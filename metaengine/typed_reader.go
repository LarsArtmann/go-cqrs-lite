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
			return zero, false, fmt.Errorf("coalescer: unexpected result type %T", result)
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
func (r *TypedReader[V]) Scan(ctx context.Context, opts ...ScanOption) ([]V, error) {
	if err := r.store.IsPoisoned(r.collection); err != nil {
		return nil, err
	}

	cfg := scanConfig{limit: 100}

	for _, opt := range opts {
		opt(&cfg)
	}

	// PrefetchCache: serve from cache when a cursor key matches.
	if r.prefetch != nil && cfg.cursor != nil {
		cacheKey := prefetchKey(r.collection, cfg.cursor)
		if cached := r.prefetch.Get(cacheKey); cached != nil {
			result := make([]V, 0, len(cached))

			for _, row := range cached {
				v, err := reify[V](row)
				if err != nil {
					return nil, fmt.Errorf("prefetch decode %s: %w", r.collection, err)
				}

				result = append(result, v)
			}

			return trimToLimit(result, cfg.limit), nil
		}
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

	// OR groups and multi-column sort require the closure path — they
	// cannot be expressed as a single FilterSpec or SortSpec for SQL pushdown.
	needsClosure := len(cfg.orGroups) > 0 || len(cfg.sortCols) > 1

	// Fastest path: raw scan → direct decode per row (1 JSON op instead of 3).
	if rsr, ok := eng.(RawScanReader); ok && !needsClosure {
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

		return r.trimAndCache(result, cfg), nil
	}

	// Standard pushdown scan (decoded values).
	if pushdown, ok := eng.(PushdownScan); ok && !needsClosure {
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

		return r.trimAndCache(result, cfg), nil
	}

	// Closure-based fallback (in-Go filter + sort).
	if sb, ok := eng.(ScanBackend); ok {
		var filterFn func(item any) bool

		filters := cfg.filters
		orGroups := cfg.orGroups

		if len(filters) > 0 || len(orGroups) > 0 {
			filterFn = func(item any) bool {
				if !passesFilterSpecs(item, filters) {
					return false
				}

				for _, group := range orGroups {
					matchFound := false

					for _, spec := range group {
						if evalFilterOp(spec.Op, itemFieldByName(item, spec.Column), spec.Value) {
							matchFound = true

							break
						}
					}

					if !matchFound {
						return false
					}
				}

				return true
			}
		}

		var sortFn func(a, b any) int

		if len(cfg.sortCols) > 1 {
			cols := cfg.sortCols
			sortFn = func(a, b any) int {
				for _, col := range cols {
					c := compareValue(
						itemFieldByName(a, col.Column),
						itemFieldByName(b, col.Column),
					)
					if c != 0 {
						if col.Desc {
							return -c
						}

						return c
					}
				}

				return 0
			}
		} else if cfg.sort != nil {
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

		return r.trimAndCache(result, cfg), nil
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
func (r *TypedReader[V]) Sum(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateSum, column, opts...)
}

// Min returns the minimum value of a column across matching rows.
func (r *TypedReader[V]) Min(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateMin, column, opts...)
}

// Max returns the maximum value of a column across matching rows.
func (r *TypedReader[V]) Max(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) (float64, error) {
	return r.aggregatePushdown(ctx, AggregateMax, column, opts...)
}

// Avg returns the average value of a column across matching rows.
func (r *TypedReader[V]) Avg(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) (float64, error) {
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
		cfg.filters = append(
			cfg.filters,
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

// GroupBy scans matching rows and groups them by the value of a column.
// Returns a map from column value to the slice of values in that group.
// Uses Scan + Go-side grouping (no SQL GROUP BY pushdown yet).
func (r *TypedReader[V]) GroupBy(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) (map[any][]V, error) {
	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, err
	}

	result := make(map[any][]V, len(rows))

	for _, row := range rows {
		key := extractValueByName(row, column)
		result[key] = append(result[key], row)
	}

	return result, nil
}

// --- Scan options ---

type scanConfig struct {
	filters  []FilterSpec
	orGroups [][]FilterSpec
	sort     *SortSpec
	sortCols []SortColumn
	cursor   any
	limit    int
	ranges   []RangeSpec
	inSpecs  []InSpec
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

// SortColumn declares one column in a compound sort.
type SortColumn struct {
	Column string
	Desc   bool
}

// WithSortColumns sets a compound sort (multi-column ORDER BY). When set,
// single-column WithSort is ignored. Pushdown engines use only the first
// column; the full multi-column sort is applied in the closure fallback path.
func WithSortColumns(cols ...SortColumn) ScanOption {
	return func(c *scanConfig) {
		c.sortCols = cols
		if len(cols) > 0 {
			c.sort = &SortSpec{Column: cols[0].Column, Desc: cols[0].Desc}
		}
	}
}

// WithOr adds an OR group of filters. Each call adds one parenthesized OR
// group: at least one filter in the group must match. Multiple WithOr calls
// are ANDed together (and with WithFilter conditions).
//
// On pushdown engines, OR groups generate SQL: AND (cond1 OR cond2 ...).
// On closure-based engines, they are evaluated in Go.
func WithOr(filters ...FilterSpec) ScanOption {
	return func(c *scanConfig) {
		if len(filters) > 0 {
			c.orGroups = append(c.orGroups, filters)
		}
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

// ScanPage performs a scan and returns both the results and the next-page
// cursor for keyset pagination. The cursor is derived from the sort field of
// the last returned item. Pass it back via WithCursor(cursor.Value) on the
// next call, or use cursor.Encode() for an HTTP-safe opaque string.
//
// When a PrefetchCache is attached, ScanPage auto-populates it: extra rows
// beyond the limit are cached so the next page request is served from cache
// instead of hitting the engine.
//
// Returns (items, nil cursor, nil) when there are no more pages.
func (r *TypedReader[V]) ScanPage(ctx context.Context, opts ...ScanOption) ([]V, *Cursor, error) {
	result, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	cfg := scanConfig{limit: 100}
	for _, opt := range opts {
		opt(&cfg)
	}

	if len(result) == 0 || cfg.limit <= 0 || len(result) < cfg.limit {
		return result, nil, nil
	}

	nextVal := extractCursorValue(result[cfg.limit-1], cfg)

	return result, &Cursor{Value: nextVal}, nil
}

// trimAndCache trims results to the limit and, when a PrefetchCache is attached,
// caches the extra rows beyond the limit for the next page. The next page's
// cursor key is derived from the last returned item's sort field (or the item
// itself when no sort is specified).
func (r *TypedReader[V]) trimAndCache(result []V, cfg scanConfig) []V {
	if r.prefetch != nil && cfg.limit > 0 && len(result) > cfg.limit {
		cursorVal := extractCursorValue(result[cfg.limit-1], cfg)
		nextKey := prefetchKey(r.collection, cursorVal)
		extra := make([]any, len(result)-cfg.limit)

		for i, v := range result[cfg.limit:] {
			extra[i] = v
		}

		r.prefetch.Put(nextKey, extra)
	}

	return trimToLimit(result, cfg.limit)
}

// extractCursorValue derives the raw cursor value from an item using the scan
// config's sort specification. When a sort spec is set, the cursor is the sort
// column value; otherwise the whole item is used.
func extractCursorValue[V any](item V, cfg scanConfig) any {
	if cfg.sort != nil {
		return itemFieldByName(item, cfg.sort.Column)
	}

	if len(cfg.sortCols) > 0 {
		return itemFieldByName(item, cfg.sortCols[0].Column)
	}

	return item
}

// prefetchKey builds the PrefetchCache lookup key from a collection name and
// cursor value. Both trimAndCache (cache write) and the prefetch check in Scan
// (cache read) use this function, ensuring the key formats always match.
func prefetchKey(collection string, cursorVal any) string {
	return fmt.Sprintf("%s:%v", collection, cursorVal)
}

// trimToLimit trims the result to the limit (engines return limit+1 rows for
// has-more detection, which TypedReader.Scan discards).
func trimToLimit[V any](result []V, limit int) []V {
	if limit > 0 && len(result) > limit {
		return result[:limit]
	}

	return result
}
