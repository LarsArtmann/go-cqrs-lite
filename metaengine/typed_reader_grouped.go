package metaengine

import (
	"context"
	"fmt"
)

// Distinct returns the unique values of a column across matching rows.
// When the engine implements DistinctReader, the dedup is pushed into SQL
// (SELECT DISTINCT) — zero rows loaded for dedup. Otherwise falls back to
// Scan + Go-side dedup.
func (r *TypedReader[V]) Distinct(
	ctx context.Context,
	column string,
	opts ...ScanOption,
) ([]any, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if dr, ok := eng.(DistinctReader); ok {
			return dr.DistinctValues(ctx, r.collection, column, filters) //nolint:wrapcheck
		}
	}

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
// This method returns the actual rows per group (not aggregate counts).
// For aggregate pushdown (COUNT/SUM per group), use GroupedCount/GroupedSum/etc.
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

// --- SQL GROUP BY pushdown methods (GroupedAggregateReader) ---
//
// These methods push GROUP BY + aggregate into SQL when the engine supports
// it (DuckDB's vectorized execution is the primary beneficiary). They return
// map[string]float64 (group key → aggregate value), not the rows themselves.
// When the engine doesn't implement GroupedAggregateReader, they fall back
// to Scan + Go-side aggregation.

// GroupedCount counts rows per group, pushed down to SQL GROUP BY when available.
func (r *TypedReader[V]) GroupedCount(
	ctx context.Context,
	groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	return r.groupedAggregatePushdown(ctx, AggregateCount, "", groupBy, opts...)
}

// GroupedSum sums a column per group, pushed down to SQL GROUP BY when available.
func (r *TypedReader[V]) GroupedSum(
	ctx context.Context,
	column, groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	return r.groupedAggregatePushdown(ctx, AggregateSum, column, groupBy, opts...)
}

// GroupedMin finds the minimum of a column per group, pushed down when available.
func (r *TypedReader[V]) GroupedMin(
	ctx context.Context,
	column, groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	return r.groupedAggregatePushdown(ctx, AggregateMin, column, groupBy, opts...)
}

// GroupedMax finds the maximum of a column per group, pushed down when available.
func (r *TypedReader[V]) GroupedMax(
	ctx context.Context,
	column, groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	return r.groupedAggregatePushdown(ctx, AggregateMax, column, groupBy, opts...)
}

// GroupedAvg averages a column per group, pushed down to SQL GROUP BY when available.
func (r *TypedReader[V]) GroupedAvg(
	ctx context.Context,
	column, groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	return r.groupedAggregatePushdown(ctx, AggregateAvg, column, groupBy, opts...)
}

func (r *TypedReader[V]) groupedAggregatePushdown(
	ctx context.Context,
	fn AggregateFn,
	column, groupBy string,
	opts ...ScanOption,
) (map[string]float64, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if gr, ok := eng.(GroupedAggregateReader); ok {
			return gr.GroupedAggregate(
				ctx,
				r.collection,
				fn,
				column,
				groupBy,
				filters,
			) //nolint:wrapcheck
		}
	}

	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)

	for _, row := range rows {
		key := fmt.Sprint(extractValueByName(row, groupBy))

		switch fn {
		case AggregateCount:
			result[key]++
		case AggregateSum, AggregateAvg:
			if n, ok := toFloat64(extractValueByName(row, column)); ok {
				result[key] += n
			}
		case AggregateMin:
			if n, ok := toFloat64(extractValueByName(row, column)); ok {
				if existing, exists := result[key]; !exists || n < existing {
					result[key] = n
				}
			}
		case AggregateMax:
			if n, ok := toFloat64(extractValueByName(row, column)); ok {
				if existing, exists := result[key]; !exists || n > existing {
					result[key] = n
				}
			}
		}
	}

	if fn == AggregateAvg {
		counts := make(map[string]int)
		for _, row := range rows {
			key := fmt.Sprint(extractValueByName(row, groupBy))
			counts[key]++
		}

		for key, total := range result {
			if c := counts[key]; c > 0 {
				result[key] = total / float64(c)
			}
		}
	}

	return result, nil
}

// --- Multi-aggregate pushdown (MultiAggregateReader) ---
//
// MultiAggregate computes multiple scalar aggregates in a single SQL pass.
// DuckDB's vectorized engine computes COUNT, SUM, AVG, MIN, MAX simultaneously
// in one columnar scan — N queries collapsed to 1.

// MultiAggregate computes multiple aggregates in one query. Each spec defines
// a function + column pair; the result map is keyed by each spec's Alias
// (or a default name). Pushed down to SQL when the engine supports it.
func (r *TypedReader[V]) MultiAggregate(
	ctx context.Context,
	specs []AggregateSpec,
	opts ...ScanOption,
) (map[string]float64, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if mr, ok := eng.(MultiAggregateReader); ok {
			return mr.MultiAggregate(ctx, r.collection, specs, filters) //nolint:wrapcheck
		}
	}

	result := make(map[string]float64, len(specs))

	for _, s := range specs {
		val, err := r.aggregatePushdown(ctx, s.Fn, s.Column, opts...)
		if err != nil {
			return nil, err
		}

		result[s.AliasOr()] = val
	}

	return result, nil
}

// MultiGroupedAggregate computes multiple aggregates grouped by a column in
// one SQL pass. Each returned row has the group key plus a Values map keyed
// by each spec's Alias. Pushed down when the engine supports it.
func (r *TypedReader[V]) MultiGroupedAggregate(
	ctx context.Context,
	specs []AggregateSpec,
	groupBy string,
	opts ...ScanOption,
) ([]GroupedAggregateRow, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if mgr, ok := eng.(MultiGroupedAggregateReader); ok {
			return mgr.MultiGroupedAggregate(
				ctx,
				r.collection,
				specs,
				groupBy,
				filters,
			) //nolint:wrapcheck
		}
	}

	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, err
	}

	type groupAccum struct {
		vals          map[string]float64
		nonNullCounts map[string]int
	}

	groups := make(map[string]*groupAccum)

	for _, row := range rows {
		key := fmt.Sprint(extractValueByName(row, groupBy))

		acc, exists := groups[key]
		if !exists {
			acc = &groupAccum{
				vals:          make(map[string]float64, len(specs)),
				nonNullCounts: make(map[string]int, len(specs)),
			}
			groups[key] = acc
		}

		for _, s := range specs {
			alias := s.AliasOr()

			switch s.Fn {
			case AggregateCount:
				acc.vals[alias]++
			case AggregateSum, AggregateAvg:
				if n, ok := toFloat64(extractValueByName(row, s.Column)); ok {
					acc.vals[alias] += n
					if s.Fn == AggregateAvg {
						acc.nonNullCounts[alias]++
					}
				}
			case AggregateMin:
				if n, ok := toFloat64(extractValueByName(row, s.Column)); ok {
					if existing, ok := acc.vals[alias]; !ok || n < existing {
						acc.vals[alias] = n
					}
				}
			case AggregateMax:
				if n, ok := toFloat64(extractValueByName(row, s.Column)); ok {
					if existing, ok := acc.vals[alias]; !ok || n > existing {
						acc.vals[alias] = n
					}
				}
			}
		}
	}

	result := make([]GroupedAggregateRow, 0, len(groups))
	for key, acc := range groups {
		values := make(map[string]float64, len(specs))
		for _, s := range specs {
			alias := s.AliasOr()
			if s.Fn == AggregateAvg {
				if c := acc.nonNullCounts[alias]; c > 0 {
					values[alias] = acc.vals[alias] / float64(c)
				}
				continue
			}
			values[alias] = acc.vals[alias]
		}

		result = append(result, GroupedAggregateRow{Group: key, Values: values})
	}

	return result, nil
}
