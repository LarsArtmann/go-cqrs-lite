package sqliteengine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// SQLite implementations of the grouped/multi/distinct aggregate interfaces.
// These mirror the DuckDB engine's implementations but use SQLite placeholder
// syntax (?) and rely on SQLite's json_extract returning native types (no
// CAST AS DOUBLE needed).
//art-dupl:accept cross-module SQL builder pattern — separate go.mod

// ---------------------------------------------------------------------------
// GroupedAggregateReader (GROUP BY + single aggregate)
// ---------------------------------------------------------------------------

func (e *sqliteEngine) GroupedAggregate(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	if plan, ok := e.plans[col]; ok {
		return e.groupedAggregatePlanned(ctx, plan, fn, column, groupBy, filters)
	}

	return e.groupedAggregateStandard(ctx, col, fn, column, groupBy, filters)
}

func (e *sqliteEngine) groupedAggregateStandard(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := []any{col}

	grpExpr := fmt.Sprintf("json_extract(value, '%s')", jsonPath(groupBy))
	aggPart := aggExprSQLite(fn, column, false)

	fmt.Fprintf(&b, `SELECT %s AS group_key, %s AS agg_val FROM meta_map WHERE collection = ?`,
		grpExpr, aggPart)

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	b.WriteString(" GROUP BY group_key")

	return e.scanGroupedSQLite(ctx, b.String(), args)
}

func (e *sqliteEngine) groupedAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := []any{}

	grpExpr := metaengine.QuoteIdent(groupBy)
	aggPart := aggExprSQLite(fn, column, true)

	fmt.Fprintf(&b, "SELECT %s AS group_key, %s AS agg_val FROM %s",
		grpExpr, aggPart, metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	b.WriteString(" GROUP BY group_key")

	return e.scanGroupedSQLite(ctx, b.String(), args)
}

func (e *sqliteEngine) scanGroupedSQLite(
	ctx context.Context,
	query string,
	args []any,
) (map[string]float64, error) {
	rows, err := e.xd().QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.GroupedAggregate: %w", err)
	}

	defer metaengine.DeferClose(rows)

	result := make(map[string]float64)

	for rows.Next() {
		var key string

		var val float64

		if err := rows.Scan(&key, &val); err != nil {
			return nil, fmt.Errorf("sqliteengine.GroupedAggregate: scan: %w", err)
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("sqliteengine.GroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// MultiAggregateReader (multiple scalar aggregates in one pass)
// ---------------------------------------------------------------------------

func (e *sqliteEngine) MultiAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	if len(specs) == 0 {
		return nil, errors.New("sqliteengine.MultiAggregate: no specs provided")
	}

	if plan, ok := e.plans[col]; ok {
		return e.multiAggregatePlanned(ctx, plan, specs, filters)
	}

	return e.multiAggregateStandard(ctx, col, specs, filters)
}

func (e *sqliteEngine) multiAggregateStandard(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := []any{col}

	selectCols := make([]string, len(specs))
	for i, s := range specs {
		selectCols[i] = fmt.Sprintf("%s AS %s",
			aggExprSQLite(s.Fn, s.Column, false),
			metaengine.QuoteIdent(s.AliasOr()))
	}

	fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = ?",
		strings.Join(selectCols, ", "))

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	return e.scanMultiSQLite(ctx, b.String(), args, specs)
}

func (e *sqliteEngine) multiAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := []any{}

	selectCols := make([]string, len(specs))
	for i, s := range specs {
		selectCols[i] = fmt.Sprintf("%s AS %s",
			aggExprSQLite(s.Fn, s.Column, true),
			metaengine.QuoteIdent(s.AliasOr()))
	}

	fmt.Fprintf(&b, "SELECT %s FROM %s",
		strings.Join(selectCols, ", "),
		metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	return e.scanMultiSQLite(ctx, b.String(), args, specs)
}

func (e *sqliteEngine) scanMultiSQLite(
	ctx context.Context,
	query string,
	args []any,
	specs []metaengine.AggregateSpec,
) (map[string]float64, error) {
	raws := make([]any, len(specs))
	ptrs := make([]any, len(specs))
	for i := range raws {
		ptrs[i] = &raws[i]
	}

	if err := e.xd().QueryRowContext(ctx, query, args...).Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("sqliteengine.MultiAggregate: %w", err)
	}

	return metaengine.DecodeFloatResults(raws, specs, "sqliteengine.MultiAggregate")
}

// ---------------------------------------------------------------------------
// MultiGroupedAggregateReader (GROUP BY + multiple aggregates)
// ---------------------------------------------------------------------------

func (e *sqliteEngine) MultiGroupedAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	if len(specs) == 0 {
		return nil, errors.New("sqliteengine.MultiGroupedAggregate: no specs provided")
	}

	if plan, ok := e.plans[col]; ok {
		return e.multiGroupedAggregatePlanned(ctx, plan, specs, groupBy, filters)
	}

	return e.multiGroupedAggregateStandard(ctx, col, specs, groupBy, filters)
}

func (e *sqliteEngine) multiGroupedAggregateStandard(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	var b strings.Builder

	args := []any{col}

	grpExpr := fmt.Sprintf("json_extract(value, '%s')", jsonPath(groupBy))

	selectCols := make([]string, 0, 1+len(specs))
	selectCols = append(selectCols, grpExpr+" AS group_key")
	for _, s := range specs {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s",
			aggExprSQLite(s.Fn, s.Column, false),
			metaengine.QuoteIdent(s.AliasOr())))
	}

	fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = ?",
		strings.Join(selectCols, ", "))

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	b.WriteString(" GROUP BY group_key")

	return e.scanMultiGroupedSQLite(ctx, b.String(), args, specs)
}

func (e *sqliteEngine) multiGroupedAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	var b strings.Builder

	args := []any{}

	grpExpr := metaengine.QuoteIdent(groupBy)

	selectCols := make([]string, 0, 1+len(specs))
	selectCols = append(selectCols, grpExpr+" AS group_key")
	for _, s := range specs {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s",
			aggExprSQLite(s.Fn, s.Column, true),
			metaengine.QuoteIdent(s.AliasOr())))
	}

	fmt.Fprintf(&b, "SELECT %s FROM %s",
		strings.Join(selectCols, ", "),
		metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	b.WriteString(" GROUP BY group_key")

	return e.scanMultiGroupedSQLite(ctx, b.String(), args, specs)
}

func (e *sqliteEngine) scanMultiGroupedSQLite(
	ctx context.Context,
	query string,
	args []any,
	specs []metaengine.AggregateSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	rows, err := e.xd().QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.MultiGroupedAggregate: %w", err)
	}

	defer metaengine.DeferClose(rows)

	var result []metaengine.GroupedAggregateRow

	for rows.Next() {
		var groupKey string

		raws := make([]any, len(specs))
		scanTargets := make([]any, 0, 1+len(specs))
		scanTargets = append(scanTargets, &groupKey)

		for i := range raws {
			scanTargets = append(scanTargets, &raws[i])
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("sqliteengine.MultiGroupedAggregate: scan: %w", err)
		}

		values := make(map[string]float64, len(specs))
		for i, s := range specs {
			val, err := metaengine.DecodeFloat(raws[i])
			if err != nil {
				return nil, fmt.Errorf("sqliteengine.MultiGroupedAggregate alias %q: %w",
					s.AliasOr(), err)
			}

			values[s.AliasOr()] = val
		}

		result = append(result, metaengine.GroupedAggregateRow{Group: groupKey, Values: values})
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("sqliteengine.MultiGroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// DistinctReader (SELECT DISTINCT pushdown)
// ---------------------------------------------------------------------------

func (e *sqliteEngine) DistinctValues(
	ctx context.Context,
	col string,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	if plan, ok := e.plans[col]; ok {
		return e.distinctPlanned(ctx, plan, column, filters)
	}

	return e.distinctStandard(ctx, col, column, filters)
}

func (e *sqliteEngine) distinctStandard(
	ctx context.Context,
	col string,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	var b strings.Builder

	args := []any{col}

	fmt.Fprintf(
		&b,
		`SELECT DISTINCT json_extract(value, '%s') AS dv FROM meta_map WHERE collection = ?`,
		jsonPath(column),
	)

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	return metaengine.ScanDistinctValues(ctx, e.xd(), b.String(), args, "sqliteengine.DistinctValues")
}

func (e *sqliteEngine) distinctPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT DISTINCT %s AS dv FROM %s",
		metaengine.QuoteIdent(column), metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	return metaengine.ScanDistinctValues(ctx, e.xd(), b.String(), args, "sqliteengine.DistinctValues")
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// aggExprSQLite builds the SQL aggregate expression for SQLite. Unlike DuckDB,
// SQLite's json_extract returns native types, so no CAST is needed.
func aggExprSQLite(fn metaengine.AggregateFn, column string, planned bool) string {
	if fn == metaengine.AggregateCount {
		return "COUNT(*)"
	}

	if planned {
		return fmt.Sprintf("%s(%s)", fn, metaengine.QuoteIdent(column))
	}

	return fmt.Sprintf("%s(json_extract(value, '%s'))", fn, jsonPath(column))
}

// Compile-time assertions for the new interfaces.
var (
	_ metaengine.GroupedAggregateReader      = (*sqliteEngine)(nil)
	_ metaengine.MultiAggregateReader        = (*sqliteEngine)(nil)
	_ metaengine.MultiGroupedAggregateReader = (*sqliteEngine)(nil)
	_ metaengine.DistinctReader              = (*sqliteEngine)(nil)
)
