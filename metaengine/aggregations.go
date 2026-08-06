package metaengine

import "context"

// AggregateFn is a SQL aggregate function name.
type AggregateFn string

const (
	AggregateCount AggregateFn = "COUNT"
	AggregateSum   AggregateFn = "SUM"
	AggregateMin   AggregateFn = "MIN"
	AggregateMax   AggregateFn = "MAX"
	AggregateAvg   AggregateFn = "AVG"
)

// AggregateReader is an optional capability: engines that support SQL-level
// aggregation (COUNT, SUM, MIN, MAX, AVG) implement this to avoid loading
// all rows into Go memory. TypedReader.Count/Sum/Min/Max/Avg prefer this
// interface when available.
type AggregateReader interface {
	Aggregate(
		ctx context.Context,
		col string,
		fn AggregateFn,
		column string,
		filters []FilterSpec,
	) (float64, error)
}

