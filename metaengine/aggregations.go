package metaengine

import (
	"context"
	"fmt"
	"strings"
)

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

// Aggregate pushes the aggregate function into SQL, returning a single scalar.
// For COUNT, column is ignored (COUNT(*) is used). For SUM/MIN/MAX/AVG, the
// column determines which JSON field (standard) or table column (planned) to
// aggregate on.
func (e *sqliteEngine) Aggregate(
	ctx context.Context,
	col string,
	fn AggregateFn,
	column string,
	filters []FilterSpec,
) (float64, error) {
	if plan, ok := e.plans[col]; ok {
		return e.aggregatePlanned(ctx, plan, fn, column, filters)
	}

	return e.aggregateStandard(ctx, col, fn, column, filters)
}

func (e *sqliteEngine) aggregateStandard(
	ctx context.Context,
	col string,
	fn AggregateFn,
	column string,
	filters []FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := []any{col}

	if fn == AggregateCount {
		b.WriteString(`SELECT COUNT(*) FROM meta_map WHERE collection = ?`)
	} else {
		path := jsonPath(column)
		fmt.Fprintf(&b, `SELECT %s(json_extract(value, '%s')) FROM meta_map WHERE collection = ?`,
			fn,
			path)
	}

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	var result float64

	if fn == AggregateCount {
		var count int64
		if err := e.xd().QueryRowContext(ctx, b.String(), args...).Scan(&count); err != nil {
			return 0, fmt.Errorf("aggregate %s count: %w", col, err)
		}

		return float64(count), nil
	}

	if err := e.xd().QueryRowContext(ctx, b.String(), args...).Scan(&result); err != nil {
		return 0, fmt.Errorf("aggregate %s %s(%s): %w", col, fn, column, err)
	}

	return result, nil
}

func (e *sqliteEngine) aggregatePlanned(
	ctx context.Context,
	plan LayoutPlan,
	fn AggregateFn,
	column string,
	filters []FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := []any{}

	if fn == AggregateCount {
		fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", quoteIdent(plan.Table))
	} else {
		fmt.Fprintf(&b, "SELECT %s(%s) FROM %s", fn, quoteIdent(column), quoteIdent(plan.Table))
	}

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	var result float64

	if fn == AggregateCount {
		var count int64
		if err := e.xd().QueryRowContext(ctx, b.String(), args...).Scan(&count); err != nil {
			return 0, fmt.Errorf("aggregate %s count: %w", plan.Collection, err)
		}

		return float64(count), nil
	}

	if err := e.xd().QueryRowContext(ctx, b.String(), args...).Scan(&result); err != nil {
		return 0, fmt.Errorf("aggregate %s %s(%s): %w", plan.Collection, fn, column, err)
	}

	return result, nil
}
