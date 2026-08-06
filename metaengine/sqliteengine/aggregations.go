package sqliteengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Aggregate pushes the aggregate function into SQL, returning a single scalar.
func (e *sqliteEngine) Aggregate(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	if plan, ok := e.plans[col]; ok {
		return e.aggregatePlanned(ctx, plan, fn, column, filters)
	}

	return e.aggregateStandard(ctx, col, fn, column, filters)
}

func (e *sqliteEngine) aggregateStandard(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := []any{col}

	if fn == metaengine.AggregateCount {
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

	if fn == metaengine.AggregateCount {
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
	plan metaengine.LayoutPlan,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := []any{}

	if fn == metaengine.AggregateCount {
		fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", metaengine.QuoteIdent(plan.Table))
	} else {
		fmt.Fprintf(
			&b,
			"SELECT %s(%s) FROM %s",
			fn,
			metaengine.QuoteIdent(column),
			metaengine.QuoteIdent(plan.Table),
		)
	}

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	var result float64

	if fn == metaengine.AggregateCount {
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
