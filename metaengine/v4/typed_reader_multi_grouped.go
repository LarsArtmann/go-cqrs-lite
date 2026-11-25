package metaengine

import (
	"context"
	"fmt"
)

// MultiGroupedAggregate computes multiple aggregates grouped by a column in
// one SQL pass. Each returned row has the group key plus a Values map keyed
// by each spec's Alias. Pushed down when the engine supports it.
func (r *TypedReader[V]) MultiGroupedAggregate(
	ctx context.Context,
	specs []AggregateSpec,
	groupBy string,
	opts ...ScanOption,
) ([]GroupedAggregateRow, error) {
	filters, err := r.aggregateFilters(opts...)
	if err != nil {
		return nil, err
	}

	err = validateMultiAggregate(specs, groupBy)
	if err != nil {
		return nil, err
	}

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
