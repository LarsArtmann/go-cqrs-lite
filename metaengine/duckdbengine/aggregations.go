package duckdbengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DuckDB's vectorized columnar execution engine makes aggregate queries
// (COUNT, SUM, AVG, MIN, MAX, GROUP BY, DISTINCT) its killer feature.
// This file pushes all aggregate computations into DuckDB SQL — zero rows
// are loaded into Go memory for scalar/grouped/multi aggregates.
//
// Two paths exist for every method:
//   - Standard path: json_extract(value, '$.field') on meta_map.
//   - Planned path: direct column references on dedicated planned tables
//     (created via ApplyLayout). Planned tables enable DuckDB's zone maps
//     and ART indexes for even faster aggregation.

// ---------------------------------------------------------------------------
// Shared SQL builders
// ---------------------------------------------------------------------------

// aggExpr builds the SQL expression for an aggregate function applied to a
// column. On the standard path, it uses json_extract; on the planned path,
// it uses direct column references. COUNT(*) ignores column.
func aggExpr(fn metaengine.AggregateFn, column string, plan metaengine.LayoutPlan) string {
	if fn == metaengine.AggregateCount {
		return "COUNT(*)"
	}

	if plan.Table != "" {
		return fmt.Sprintf("%s(%s)", fn, metaengine.QuoteIdent(column))
	}

	return fmt.Sprintf("%s(json_extract(value, '%s'))", fn, jsonPath(column))
}

// groupExpr builds the SQL expression for the GROUP BY column.
func groupExpr(column string, plan metaengine.LayoutPlan) string {
	if plan.Table != "" {
		return metaengine.QuoteIdent(column)
	}

	return fmt.Sprintf("json_extract(value, '%s')", jsonPath(column))
}

// columnExpr builds the SQL expression for a value column used in DISTINCT
// or filter contexts.
func columnExpr(column string, plan metaengine.LayoutPlan) string {
	if plan.Table != "" {
		return metaengine.QuoteIdent(column)
	}

	return fmt.Sprintf("json_extract(value, '%s')", jsonPath(column))
}

// appendDuckDBFilter adds one filter clause to the builder for DuckDB SQL.
// On the standard path, uses json_extract + $N::json; on the planned path,
// uses direct column + $N.
func appendDuckDBFilter(
	b *strings.Builder,
	args *[]any,
	argIdx *int,
	f metaengine.FilterSpec,
	plan metaengine.LayoutPlan,
) {
	if f.Op == metaengine.FilterIn {
		values, ok := f.Value.([]any)
		if !ok || len(values) == 0 {
			return
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			if plan.Table != "" {
				placeholders[i] = fmt.Sprintf("$%d", *argIdx)
				*args = append(*args, v)
			} else {
				jb, _ := json.Marshal(v)
				placeholders[i] = fmt.Sprintf("$%d::json", *argIdx)
				*args = append(*args, string(jb))
			}

			*argIdx++
		}

		fmt.Fprintf(b, " AND %s IN (%s)", columnExpr(f.Column, plan), strings.Join(placeholders, ", "))
	} else {
		if plan.Table != "" {
			fmt.Fprintf(b, " AND %s %s $%d", columnExpr(f.Column, plan), string(f.Op), *argIdx)
			*args = append(*args, f.Value)
		} else {
			jb, _ := json.Marshal(f.Value)
			fmt.Fprintf(b, " AND %s %s $%d::json", columnExpr(f.Column, plan), string(f.Op), *argIdx)
			*args = append(*args, string(jb))
		}

		*argIdx++
	}
}

// fromClause returns the FROM clause + initial WHERE collection filter.
func fromClause(col string, plan metaengine.LayoutPlan) string {
	if plan.Table != "" {
		return fmt.Sprintf("FROM %s", metaengine.QuoteIdent(plan.Table))
	}

	return fmt.Sprintf("FROM meta_map WHERE collection = $1")
}

// initialArgIndex returns the starting arg index ($N). Standard path uses $1
// for the collection filter; planned path starts at $1 with no collection filter.
func initialArgIndex(plan metaengine.LayoutPlan) int {
	if plan.Table != "" {
		return 1
	}

	return 2 // $1 is collection
}

// initialArgs returns the initial args slice. Standard path includes the
// collection name; planned path is empty.
func initialArgs(col string, plan metaengine.LayoutPlan) []any {
	if plan.Table != "" {
		return []any{}
	}

	return []any{col}
}

// ---------------------------------------------------------------------------
// AggregateReader (scalar aggregates: COUNT, SUM, MIN, MAX, AVG)
// ---------------------------------------------------------------------------

func (e *duckdbEngine) Aggregate(
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

func (e *duckdbEngine) aggregateStandard(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := initialArgs(col, metaengine.LayoutPlan{})
	argIdx := initialArgIndex(metaengine.LayoutPlan{})

	fmt.Fprintf(&b, "SELECT %s %s", aggExpr(fn, column, metaengine.LayoutPlan{}), fromClause(col, metaengine.LayoutPlan{}))

	for _, f := range filters {
		appendDuckDBFilter(&b, &args, &argIdx, f, metaengine.LayoutPlan{})
	}

	return e.scanScalar(ctx, b.String(), args, col, fn, column)
}

func (e *duckdbEngine) aggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := initialArgs(plan.Collection, plan)
	argIdx := initialArgIndex(plan)

	fmt.Fprintf(&b, "SELECT %s %s", aggExpr(fn, column, plan), fromClause(plan.Collection, plan))

	whereStarted := false

	for _, f := range filters {
		if !whereStarted {
			b.WriteString(" WHERE ")
			whereStarted = true
		}

		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	return e.scanScalar(ctx, b.String(), args, plan.Collection, fn, column)
}

func (e *duckdbEngine) scanScalar(
	ctx context.Context,
	query string,
	args []any,
	col string,
	fn metaengine.AggregateFn,
	column string,
) (float64, error) {
	var raw any

	if err := e.conn().QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		return 0, fmt.Errorf("duckdbengine.Aggregate %s %s(%s): %w", col, fn, column, err)
	}

	return decodeFloat(raw)
}

// decodeFloat converts a DuckDB scalar scan value to float64. DuckDB returns
// float64 for SUM/AVG, int64 for COUNT, and may return nil for empty sets.
func decodeFloat(raw any) (float64, error) {
	if raw == nil {
		return 0, nil
	}

	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case []byte:
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			return 0, fmt.Errorf("duckdbengine decodeFloat: %w", err)
		}

		return f, nil
	default:
		return 0, fmt.Errorf("duckdbengine decodeFloat: unexpected type %T", raw)
	}
}

// ---------------------------------------------------------------------------
// GroupedAggregateReader (GROUP BY + single aggregate)
// ---------------------------------------------------------------------------

func (e *duckdbEngine) GroupedAggregate(
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

func (e *duckdbEngine) groupedAggregateStandard(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := initialArgs(col, metaengine.LayoutPlan{})
	argIdx := initialArgIndex(metaengine.LayoutPlan{})
	plan := metaengine.LayoutPlan{}

	gExpr := groupExpr(groupBy, plan)

	fmt.Fprintf(&b, "SELECT %s AS group_key, %s AS agg_val %s",
		gExpr, aggExpr(fn, column, plan), fromClause(col, plan))

	for _, f := range filters {
		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	fmt.Fprintf(&b, " GROUP BY group_key")

	return e.scanGrouped(ctx, b.String(), args)
}

func (e *duckdbEngine) groupedAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := initialArgs(plan.Collection, plan)
	argIdx := initialArgIndex(plan)

	gExpr := groupExpr(groupBy, plan)

	fmt.Fprintf(&b, "SELECT %s AS group_key, %s AS agg_val %s",
		gExpr, aggExpr(fn, column, plan), fromClause(plan.Collection, plan))

	whereStarted := false

	for _, f := range filters {
		if !whereStarted {
			b.WriteString(" WHERE ")
			whereStarted = true
		}

		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	fmt.Fprintf(&b, " GROUP BY group_key")

	return e.scanGrouped(ctx, b.String(), args)
}

func (e *duckdbEngine) scanGrouped(
	ctx context.Context,
	query string,
	args []any,
) (map[string]float64, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.GroupedAggregate: %w", err)
	}

	defer func() { _ = rows.Close() }()

	result := make(map[string]float64)

	for rows.Next() {
		var key string

		var raw any

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("duckdbengine.GroupedAggregate: scan: %w", err)
		}

		val, err := decodeFloat(raw)
		if err != nil {
			return nil, err
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("duckdbengine.GroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// MultiAggregateReader (multiple scalar aggregates in one pass)
// ---------------------------------------------------------------------------

func (e *duckdbEngine) MultiAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("duckdbengine.MultiAggregate: no specs provided")
	}

	if plan, ok := e.plans[col]; ok {
		return e.multiAggregatePlanned(ctx, plan, specs, filters)
	}

	return e.multiAggregateStandard(ctx, col, specs, filters)
}

func (e *duckdbEngine) multiAggregateStandard(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := initialArgs(col, metaengine.LayoutPlan{})
	argIdx := initialArgIndex(metaengine.LayoutPlan{})
	plan := metaengine.LayoutPlan{}

	selectCols := make([]string, len(specs))
	for i, s := range specs {
		selectCols[i] = fmt.Sprintf("%s AS %s", aggExpr(s.Fn, s.Column, plan), metaengine.QuoteIdent(s.AliasOr()))
	}

	fmt.Fprintf(&b, "SELECT %s %s", strings.Join(selectCols, ", "), fromClause(col, plan))

	for _, f := range filters {
		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	return e.scanMulti(ctx, b.String(), args, specs)
}

func (e *duckdbEngine) multiAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := initialArgs(plan.Collection, plan)
	argIdx := initialArgIndex(plan)

	selectCols := make([]string, len(specs))
	for i, s := range specs {
		selectCols[i] = fmt.Sprintf("%s AS %s", aggExpr(s.Fn, s.Column, plan), metaengine.QuoteIdent(s.AliasOr()))
	}

	fmt.Fprintf(&b, "SELECT %s %s", strings.Join(selectCols, ", "), fromClause(plan.Collection, plan))

	whereStarted := false

	for _, f := range filters {
		if !whereStarted {
			b.WriteString(" WHERE ")
			whereStarted = true
		}

		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	return e.scanMulti(ctx, b.String(), args, specs)
}

func (e *duckdbEngine) scanMulti(
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

	if err := e.conn().QueryRowContext(ctx, query, args...).Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("duckdbengine.MultiAggregate: %w", err)
	}

	result := make(map[string]float64, len(specs))
	for i, s := range specs {
		val, err := decodeFloat(raws[i])
		if err != nil {
			return nil, fmt.Errorf("duckdbengine.MultiAggregate alias %q: %w", s.AliasOr(), err)
		}

		result[s.AliasOr()] = val
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// MultiGroupedAggregateReader (GROUP BY + multiple aggregates)
// ---------------------------------------------------------------------------

func (e *duckdbEngine) MultiGroupedAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("duckdbengine.MultiGroupedAggregate: no specs provided")
	}

	if plan, ok := e.plans[col]; ok {
		return e.multiGroupedAggregatePlanned(ctx, plan, specs, groupBy, filters)
	}

	return e.multiGroupedAggregateStandard(ctx, col, specs, groupBy, filters)
}

func (e *duckdbEngine) multiGroupedAggregateStandard(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	var b strings.Builder

	args := initialArgs(col, metaengine.LayoutPlan{})
	argIdx := initialArgIndex(metaengine.LayoutPlan{})
	plan := metaengine.LayoutPlan{}

	gExpr := groupExpr(groupBy, plan)

	selectCols := []string{fmt.Sprintf("%s AS group_key", gExpr)}
	for _, s := range specs {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s", aggExpr(s.Fn, s.Column, plan), metaengine.QuoteIdent(s.AliasOr())))
	}

	fmt.Fprintf(&b, "SELECT %s %s", strings.Join(selectCols, ", "), fromClause(col, plan))

	for _, f := range filters {
		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	fmt.Fprintf(&b, " GROUP BY group_key")

	return e.scanMultiGrouped(ctx, b.String(), args, specs)
}

func (e *duckdbEngine) multiGroupedAggregatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	var b strings.Builder

	args := initialArgs(plan.Collection, plan)
	argIdx := initialArgIndex(plan)

	gExpr := groupExpr(groupBy, plan)

	selectCols := []string{fmt.Sprintf("%s AS group_key", gExpr)}
	for _, s := range specs {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s", aggExpr(s.Fn, s.Column, plan), metaengine.QuoteIdent(s.AliasOr())))
	}

	fmt.Fprintf(&b, "SELECT %s %s", strings.Join(selectCols, ", "), fromClause(plan.Collection, plan))

	whereStarted := false

	for _, f := range filters {
		if !whereStarted {
			b.WriteString(" WHERE ")
			whereStarted = true
		}

		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	fmt.Fprintf(&b, " GROUP BY group_key")

	return e.scanMultiGrouped(ctx, b.String(), args, specs)
}

func (e *duckdbEngine) scanMultiGrouped(
	ctx context.Context,
	query string,
	args []any,
	specs []metaengine.AggregateSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.MultiGroupedAggregate: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var result []metaengine.GroupedAggregateRow

	for rows.Next() {
		groupKey := ""

		raws := make([]any, len(specs))
		scanTargets := make([]any, 0, 1+len(specs))
		scanTargets = append(scanTargets, &groupKey)

		for i := range raws {
			scanTargets = append(scanTargets, &raws[i])
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("duckdbengine.MultiGroupedAggregate: scan: %w", err)
		}

		values := make(map[string]float64, len(specs))
		for i, s := range specs {
			val, err := decodeFloat(raws[i])
			if err != nil {
				return nil, fmt.Errorf("duckdbengine.MultiGroupedAggregate alias %q: %w", s.AliasOr(), err)
			}

			values[s.AliasOr()] = val
		}

		result = append(result, metaengine.GroupedAggregateRow{Group: groupKey, Values: values})
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("duckdbengine.MultiGroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// DistinctReader (SELECT DISTINCT pushdown)
// ---------------------------------------------------------------------------

func (e *duckdbEngine) DistinctValues(
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

func (e *duckdbEngine) distinctStandard(
	ctx context.Context,
	col string,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	var b strings.Builder

	args := initialArgs(col, metaengine.LayoutPlan{})
	argIdx := initialArgIndex(metaengine.LayoutPlan{})
	plan := metaengine.LayoutPlan{}

	fmt.Fprintf(&b, "SELECT DISTINCT %s AS dv %s", columnExpr(column, plan), fromClause(col, plan))

	for _, f := range filters {
		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	return e.scanDistinct(ctx, b.String(), args)
}

func (e *duckdbEngine) distinctPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	var b strings.Builder

	args := initialArgs(plan.Collection, plan)
	argIdx := initialArgIndex(plan)

	fmt.Fprintf(&b, "SELECT DISTINCT %s AS dv %s", columnExpr(column, plan), fromClause(plan.Collection, plan))

	whereStarted := false

	for _, f := range filters {
		if !whereStarted {
			b.WriteString(" WHERE ")
			whereStarted = true
		}

		appendDuckDBFilter(&b, &args, &argIdx, f, plan)
	}

	return e.scanDistinct(ctx, b.String(), args)
}

func (e *duckdbEngine) scanDistinct(
	ctx context.Context,
	query string,
	args []any,
) ([]any, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.DistinctValues: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var result []any

	for rows.Next() {
		var raw any
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("duckdbengine.DistinctValues: scan: %w", err)
		}

		result = append(result, raw)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("duckdbengine.DistinctValues: %w", err)
	}

	return result, nil
}
