package metaengine

import (
	"context"
	"fmt"
)

func (r *TypedReader[V]) scanClosure(
	ctx context.Context,
	sb ScanBackend,
	cfg scanConfig,
	fetchLimit int,
) ([]V, pageMeta, error) {
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
		return nil, pageMeta{}, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
	}

	result := make([]V, 0, len(scanResult.Items))

	for _, row := range scanResult.Items {
		v, err := reify[V](row)
		if err != nil {
			return nil, pageMeta{}, fmt.Errorf("typed reader scan %s: %w", r.collection, err)
		}

		result = append(result, v)
	}

	meta := pageMeta{nextCursor: scanResult.NextCursor, hasMore: scanResult.HasMore}
	meta.nextCursor = normalizeClosureCursor(meta.nextCursor, cfg)

	return r.trimAndCache(result, cfg), meta, nil
}

// normalizeClosureCursor narrows an engine-issued compound cursor's Sort
// component from the full last row to the bare sort-column value. Closure
// engines (memory, KV, dgraph) compare via a comparator closure and never see
// the column name, so SortPaginate cursors carry the whole row; this is the
// one place that knows cfg.sort.Column. A scalar Sort keeps the wire cursor
// small (no row leakage into HTTP-facing cursor strings) and directly
// bindable by SQL keyset predicates. Rows missing the sort field keep the
// full-row cursor (graceful degradation), and non-compound cursors pass
// through untouched.
func normalizeClosureCursor(next any, cfg scanConfig) any {
	skc, ok := next.(SortKeyCursor)
	if !ok || cfg.sort == nil {
		return next
	}

	sortVal := itemFieldByName(skc.Sort, cfg.sort.Column)
	if sortVal == nil {
		return next
	}

	return SortKeyCursor{Sort: sortVal, Key: skc.Key}
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
			av := itemFieldByName(a, col)
			bv := itemFieldByName(b, col)

			if bv == nil && b != nil && !isRowValue(b) {
				// Cursor operands arrive as bare scalar sort values (legacy
				// WithCursor cursors and SortKeyCursor.Sort after
				// normalizeClosureCursor) — there is no row to field-extract,
				// so compare against the scalar itself. Without this the
				// extraction reads nil, compareValue ranks every row
				// strictly-after the cursor, and pagination re-serves the
				// same page forever.
				bv = b
			}

			return compareValue(av, bv)
		}

		if cfg.sort.Desc {
			baseSort := sortFn

			return func(a, b any) int { return -baseSort(a, b) }
		}

		return sortFn
	}

	return nil
}
