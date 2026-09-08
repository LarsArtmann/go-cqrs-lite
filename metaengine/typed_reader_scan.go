package metaengine

import (
	"context"
	"fmt"
)

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
		cacheKey := prefetchCursorKey(r.collection, cfg.cursor)
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

	// When a PrefetchCache is attached, fetch double the limit so trimAndCache
	// can cache a full next page (not just 1 overflow row).
	fetchLimit := cfg.limit
	if r.prefetch != nil && cfg.limit > 0 {
		fetchLimit = cfg.limit * 2
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
		return r.scanRaw(ctx, rsr, cfg, fetchLimit)
	}

	// Standard pushdown scan (decoded values).
	if pushdown, ok := eng.(PushdownScan); ok && !needsClosure {
		return r.scanPushdown(ctx, pushdown, cfg, fetchLimit)
	}

	// Closure-based fallback (in-Go filter + sort).
	if sb, ok := eng.(ScanBackend); ok {
		return r.scanClosure(ctx, sb, cfg, fetchLimit)
	}

	return nil, fmt.Errorf("%w: %s", errUnsupportedScanReads, eng.Profile().Name)
}

func (r *TypedReader[V]) scanRaw(
	ctx context.Context,
	rsr RawScanReader,
	cfg scanConfig,
	fetchLimit int,
) ([]V, error) {
	rawResult, err := rsr.ScanRawValues(
		ctx, r.collection, cfg.filters, cfg.sort, rawCursorValue(cfg.cursor), fetchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
	}

	result := make([]V, 0, len(rawResult.Items))

	for _, raw := range rawResult.Items {
		v, err := reify[V](jsonValue(raw))
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result = append(result, v)
	}

	return r.trimAndCache(result, cfg), nil
}

func (r *TypedReader[V]) scanPushdown(
	ctx context.Context,
	pushdown PushdownScan,
	cfg scanConfig,
	fetchLimit int,
) ([]V, error) {
	scanResult, err := pushdown.PushdownMapScan(
		ctx, r.collection, cfg.filters, cfg.sort, rawCursorValue(cfg.cursor), fetchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
	}

	result := make([]V, 0, len(scanResult.Items))

	for _, row := range scanResult.Items {
		v, err := reify[V](row)
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result = append(result, v)
	}

	return r.trimAndCache(result, cfg), nil
}

func (r *TypedReader[V]) scanClosure(
	ctx context.Context,
	sb ScanBackend,
	cfg scanConfig,
	fetchLimit int,
) ([]V, error) {
	filterFn := buildClosureFilter(cfg)
	sortFn := buildClosureSort(cfg)

	scanResult, err := sb.MapScan(
		ctx,
		r.collection,
		filterFn,
		sortFn,
		rawCursorValue(cfg.cursor),
		fetchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
	}

	result := make([]V, 0, len(scanResult.Items))

	for _, row := range scanResult.Items {
		v, err := reify[V](row)
		if err != nil {
			return nil, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result = append(result, v)
	}

	return r.trimAndCache(result, cfg), nil
}

func buildClosureFilter(cfg scanConfig) func(item any) bool {
	filters := cfg.filters
	orGroups := cfg.orGroups

	if len(filters) == 0 && len(orGroups) == 0 {
		return nil
	}

	return func(item any) bool {
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

func buildClosureSort(cfg scanConfig) func(a, b any) int {
	if len(cfg.sortCols) > 1 {
		cols := cfg.sortCols

		return func(a, b any) int {
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
	}

	if cfg.sort != nil {
		col := cfg.sort.Column
		sortFn := func(a, b any) int {
			return compareValue(itemFieldByName(a, col), itemFieldByName(b, col))
		}

		if cfg.sort.Desc {
			baseSort := sortFn

			return func(a, b any) int { return -baseSort(a, b) }
		}

		return sortFn
	}

	return nil
}

// Exists checks whether a key is present in the collection.
// Uses SetBackend.SetContains for Set ADTs, falls back to MapGet for Map ADTs.
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
// When the engine implements DistinctReader, the dedup is pushed into SQL
// (SELECT DISTINCT) — zero rows loaded for dedup. Otherwise falls back to
// Scan + Go-side dedup.
