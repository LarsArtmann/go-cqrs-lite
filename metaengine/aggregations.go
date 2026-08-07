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

// AggregateSpec describes one aggregate computation in a multi-aggregate query.
// Column is the JSON field name (standard path) or column name (planned path).
// It is ignored for AggregateCount. Alias is the result map key; if empty it
// defaults to string(Fn) + "(" + Column + ")".
type AggregateSpec struct {
	Fn     AggregateFn
	Column string
	Alias  string
}

// Alias returns the result map key for this spec, applying the default
// naming convention when Alias is empty.
func (s AggregateSpec) AliasOr() string {
	if s.Alias != "" {
		return s.Alias
	}

	if s.Fn == AggregateCount {
		return "count"
	}

	return string(s.Fn) + "(" + s.Column + ")"
}

// GroupedAggregateRow holds one GROUP BY bucket with its aggregate values.
// Group is the string representation of the grouping column value.
type GroupedAggregateRow struct {
	Group  string
	Values map[string]float64
}

// GroupedAggregateReader pushes GROUP BY + a single aggregate function into
// SQL, returning one scalar per group. DuckDB's vectorized execution makes
// this dramatically faster than loading all rows into Go memory and grouping
// in-process.
type GroupedAggregateReader interface {
	GroupedAggregate(
		ctx context.Context,
		col string,
		fn AggregateFn,
		column string,
		groupBy string,
		filters []FilterSpec,
	) (map[string]float64, error)
}

// MultiAggregateReader computes multiple scalar aggregates in a single SQL
// pass. This is DuckDB's sweet spot: one vectorized columnar scan computes
// COUNT, SUM, AVG, MIN, MAX simultaneously instead of N separate queries.
type MultiAggregateReader interface {
	MultiAggregate(
		ctx context.Context,
		col string,
		specs []AggregateSpec,
		filters []FilterSpec,
	) (map[string]float64, error)
}

// MultiGroupedAggregateReader pushes GROUP BY + multiple aggregates into SQL.
// Example SQL: SELECT status, COUNT(*), SUM(price) FROM ... GROUP BY status.
// Each returned row has the group key plus a Values map keyed by each spec's
// Alias.
type MultiGroupedAggregateReader interface {
	MultiGroupedAggregate(
		ctx context.Context,
		col string,
		specs []AggregateSpec,
		groupBy string,
		filters []FilterSpec,
	) ([]GroupedAggregateRow, error)
}

// DistinctReader pushes SELECT DISTINCT into SQL, avoiding a full scan +
// Go-side dedup. DuckDB's columnar hash-based distinct is significantly faster
// than loading all rows.
type DistinctReader interface {
	DistinctValues(
		ctx context.Context,
		col string,
		column string,
		filters []FilterSpec,
	) ([]any, error)
}
