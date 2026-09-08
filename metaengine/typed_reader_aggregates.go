package metaengine

import "context"

func (r *TypedReader[V]) Count(ctx context.Context, opts ...ScanOption) (int, error) {
	filters := buildScanFilters(opts...)

	eng, ok := r.store.collectionEngine(r.collection)
	if ok {
		if ar, ok := eng.(AggregateReader); ok {
			n, err := ar.Aggregate(ctx, r.collection, AggregateCount, "", filters)
			if err != nil {
				return 0, err //nolint:wrapcheck
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
			return ar.Aggregate(ctx, r.collection, fn, column, filters) //nolint:wrapcheck
		}
	}

	// Fallback: in-Go aggregation via Scan.
	rows, err := r.Scan(ctx, opts...)
	if err != nil {
		return 0, err
	}

	if fn == AggregateCount {
		return float64(len(rows)), nil
	}

	var result float64

	var firstSet bool

	var nonNullCount int

	for _, row := range rows {
		val := extractValueByName(row, column)
		if n, ok := toFloat64(val); ok {
			nonNullCount++

			switch fn {
			case AggregateSum, AggregateAvg:
				result += n
			case AggregateMin:
				if !firstSet || n < result {
					result = n
					firstSet = true
				}
			case AggregateMax:
				if !firstSet || n > result {
					result = n
					firstSet = true
				}
			}
		}
	}

	if fn == AggregateAvg && nonNullCount > 0 {
		return result / float64(nonNullCount), nil
	}

	return result, nil
}

// buildScanFilters applies scan options and returns the expanded filter list
// (ranges and IN specs expanded into FilterSpecs).
