package pgengine

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Postgres implementations of all five aggregate interfaces. Postgres uses
// JSONB operators (value->'field') for column access and ::float8 casts for
// aggregate functions (Postgres cannot aggregate JSONB directly).

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// pgAggExpr builds the SQL aggregate expression for Postgres JSONB.
// COUNT(*) ignores column. Other functions cast the JSONB value to float8.
// art-dupl:accept cross-module SQL builder pattern — separate go.mod
func pgAggExpr(fn metaengine.AggregateFn, column string) string {
	if fn == metaengine.AggregateCount {
		return "COUNT(*)"
	}

	return fmt.Sprintf("%s((value->'%s')::float8)", fn, escapeJSONKey(column))
}

// appendPGFilter adds one filter clause inline (same pattern as PushdownMapScan).
// Filter type-coercion: Postgres uses value->'field' (JSONB operator) which
// preserves native JSON types; numeric comparisons use $N::jsonb parameters.
// This differs from DuckDB (CAST AS DOUBLE on json_extract) and SQLite
// (native types from json_extract, no CAST needed).
// art-dupl:accept cross-module SQL builder pattern — separate go.mod
func appendPGFilter(b *strings.Builder, args *[]any, f metaengine.FilterSpec) {
	if f.Op == metaengine.FilterIn {
		values, ok := f.Value.([]any)
		if !ok || len(values) == 0 {
			return
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			jb, _ := json.Marshal(v)
			placeholders[i] = fmt.Sprintf("$%d::jsonb", len(*args)+1)
			*args = append(*args, string(jb))
		}

		fmt.Fprintf(b, ` AND value->'%s' IN (%s)`,
			escapeJSONKey(f.Column), strings.Join(placeholders, ", "))
	} else {
		jb, _ := json.Marshal(f.Value)
		fmt.Fprintf(b, ` AND value->'%s' %s $%d::jsonb`,
			escapeJSONKey(f.Column), string(f.Op), len(*args)+1)
		*args = append(*args, string(jb))
	}
}

// ---------------------------------------------------------------------------
// AggregateReader (scalar aggregates: COUNT, SUM, MIN, MAX, AVG)
// ---------------------------------------------------------------------------

func (e *pgEngine) Aggregate(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	filters []metaengine.FilterSpec,
) (float64, error) {
	var b strings.Builder

	args := []any{col}

	fmt.Fprintf(&b, `SELECT %s FROM meta_map WHERE collection = $1`, pgAggExpr(fn, column))

	for _, f := range filters {
		appendPGFilter(&b, &args, f)
	}

	var raw any

	if err := e.conn().QueryRowContext(ctx, b.String(), args...).Scan(&raw); err != nil {
		return 0, fmt.Errorf("pgengine.Aggregate %s %s(%s): %w", col, fn, column, err)
	}

	return metaengine.DecodeFloat(raw)
}

// ---------------------------------------------------------------------------
// GroupedAggregateReader (GROUP BY + single aggregate)
// ---------------------------------------------------------------------------

func (e *pgEngine) GroupedAggregate(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
	groupBy string,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	var b strings.Builder

	args := []any{col}

	fmt.Fprintf(
		&b,
		`SELECT value->>'%s' AS group_key, %s AS agg_val FROM meta_map WHERE collection = $1`,
		escapeJSONKey(groupBy),
		pgAggExpr(fn, column),
	)

	for _, f := range filters {
		appendPGFilter(&b, &args, f)
	}

	b.WriteString(" GROUP BY group_key")

	rows, err := e.conn().QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.GroupedAggregate: %w", err)
	}

	defer metaengine.DeferClose(rows)

	result := make(map[string]float64)

	for rows.Next() {
		var key string

		var raw any

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("pgengine.GroupedAggregate: scan: %w", err)
		}

		val, err := metaengine.DecodeFloat(raw)
		if err != nil {
			return nil, err
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("pgengine.GroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// MultiAggregateReader (multiple scalar aggregates in one pass)
// ---------------------------------------------------------------------------

func (e *pgEngine) MultiAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	filters []metaengine.FilterSpec,
) (map[string]float64, error) {
	if len(specs) == 0 {
		return nil, errors.New("pgengine.MultiAggregate: no specs provided")
	}

	var b strings.Builder

	args := []any{col}

	selectCols := make([]string, len(specs))
	for i, s := range specs {
		selectCols[i] = fmt.Sprintf("%s AS %s",
			pgAggExpr(s.Fn, s.Column),
			metaengine.QuoteIdent(s.AliasOr()))
	}

	fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = $1",
		strings.Join(selectCols, ", "))

	for _, f := range filters {
		appendPGFilter(&b, &args, f)
	}

	return metaengine.MultiAggregateScan(ctx, e.conn(), b.String(), args, specs, "pgengine.MultiAggregate")
}

// ---------------------------------------------------------------------------
// MultiGroupedAggregateReader (GROUP BY + multiple aggregates)
// ---------------------------------------------------------------------------

func (e *pgEngine) MultiGroupedAggregate(
	ctx context.Context,
	col string,
	specs []metaengine.AggregateSpec,
	groupBy string,
	filters []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	if len(specs) == 0 {
		return nil, errors.New("pgengine.MultiGroupedAggregate: no specs provided")
	}

	var b strings.Builder

	args := []any{col}

	selectCols := []string{fmt.Sprintf("value->>'%s' AS group_key", escapeJSONKey(groupBy))}
	for _, s := range specs {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s",
			pgAggExpr(s.Fn, s.Column),
			metaengine.QuoteIdent(s.AliasOr())))
	}

	fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = $1",
		strings.Join(selectCols, ", "))

	for _, f := range filters {
		appendPGFilter(&b, &args, f)
	}

	b.WriteString(" GROUP BY group_key")

	rows, err := e.conn().QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.MultiGroupedAggregate: %w", err)
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
			return nil, fmt.Errorf("pgengine.MultiGroupedAggregate: scan: %w", err)
		}

		values := make(map[string]float64, len(specs))
		for i, s := range specs {
			val, err := metaengine.DecodeFloat(raws[i])
			if err != nil {
				return nil, fmt.Errorf("pgengine.MultiGroupedAggregate alias %q: %w",
					s.AliasOr(), err)
			}

			values[s.AliasOr()] = val
		}

		result = append(result, metaengine.GroupedAggregateRow{Group: groupKey, Values: values})
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("pgengine.MultiGroupedAggregate: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// DistinctReader (SELECT DISTINCT pushdown)
// ---------------------------------------------------------------------------

func (e *pgEngine) DistinctValues(
	ctx context.Context,
	col string,
	column string,
	filters []metaengine.FilterSpec,
) ([]any, error) {
	var b strings.Builder

	args := []any{col}

	fmt.Fprintf(&b, `SELECT DISTINCT value->'%s' AS dv FROM meta_map WHERE collection = $1`,
		escapeJSONKey(column))

	for _, f := range filters {
		appendPGFilter(&b, &args, f)
	}

	rows, err := e.conn().QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.DistinctValues: %w", err)
	}

	defer metaengine.DeferClose(rows)

	var result []any

	for rows.Next() {
		var raw []byte

		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("pgengine.DistinctValues: scan: %w", err)
		}

		var val any

		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("pgengine.DistinctValues: unmarshal: %w", err)
		}

		result = append(result, val)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("pgengine.DistinctValues: %w", err)
	}

	return result, nil
}

// Compile-time assertions for all five aggregate interfaces.
var (
	_ metaengine.AggregateReader             = (*pgEngine)(nil)
	_ metaengine.GroupedAggregateReader      = (*pgEngine)(nil)
	_ metaengine.MultiAggregateReader        = (*pgEngine)(nil)
	_ metaengine.MultiGroupedAggregateReader = (*pgEngine)(nil)
	_ metaengine.DistinctReader              = (*pgEngine)(nil)
)
