package mysqlengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan. It returns the SQL
// that PushdownMapScan would execute for the given collection and options,
// without running the query. MySQL uses JSON path operators (value->'$.field')
// for column access and CAST(? AS JSON) placeholders for filter values;
// MariaDB gets JSON_EXTRACT/JSON_UNQUOTE forms (see dialect.go). The SQL
// comes from the same builder PushdownMapScan executes (buildMapScanQuery).
func (e *mysqlEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) { //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	if err := metaengine.ValidateQueryIdents(opts.Filters, opts.Sort); err != nil {
		return rejectedExplain, nil
	}

	// Planned collections explain the extracted-column query (D3 slice 3):
	// EXPLAIN must reflect what actually executes.
	if plan, ok := e.planFor(collection); ok {
		q, args, err := buildPlannedScanQuery(
			plan,
			opts.Filters,
			opts.Sort,
			opts.Cursor,
			opts.Limit,
		)
		//art-dupl:accept dialect twin (pgengine explain.go) — builder diverges per dialect below
		if err != nil {
			return rejectedExplain, nil
		}

		return q, args
	}

	query, args, err := e.buildMapScanQuery(
		collection, opts.Filters, opts.Sort, opts.Cursor, opts.Limit,
	)
	if err != nil {
		return rejectedExplain, nil
	}

	return query, args
}

// ExplainAggregateQuery implements metaengine.ExplainableAggregate. It returns
// the SQL for aggregate queries without executing them.
func (e *mysqlEngine) ExplainAggregateQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainAggregateOptions,
) (string, []any) { //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	var b strings.Builder

	args := []any{collection}

	writeAggregateSelect(&b, e, opts)

	columns := []string{opts.Column, opts.Distinct, opts.GroupBy}
	for _, s := range opts.Specs {
		columns = append(columns, s.Column)
	}

	if err := metaengine.ValidateQueryIdents(opts.Filters, nil, columns...); err != nil {
		return rejectedExplain, nil
	}

	for _, f := range opts.Filters {
		e.appendMapFilter(&b, &args, f)
	}

	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

func (e *mysqlEngine) aggExpr(fn metaengine.AggregateFn, column string) string {
	field := e.jsonFieldExpr(column)

	switch fn {
	case metaengine.AggregateCount:
		return "COUNT(*)"
	case metaengine.AggregateSum:
		return fmt.Sprintf("SUM(%s)", field)
	case metaengine.AggregateMin:
		return fmt.Sprintf("MIN(%s)", field)
	case metaengine.AggregateMax:
		return fmt.Sprintf("MAX(%s)", field)
	case metaengine.AggregateAvg:
		return fmt.Sprintf("AVG(%s)", field)
	default:
		// A silent SUM default made any typo or hostile fn value LOOK like a
		// legitimate SUM answer in emitted SQL; explain must not fabricate.
		return "/* rejected: invalid aggregate fn */"
	}
}

// Compile-time assertions.
var (
	_ metaengine.ExplainableScan      = (*mysqlEngine)(nil)
	_ metaengine.ExplainableAggregate = (*mysqlEngine)(nil)
)

// writeAggregateSelect emits the SELECT clause for the three aggregate
// shapes: DISTINCT values, multi-spec grouped aggregates, and the plain
// single-aggregate form.
func writeAggregateSelect(b *strings.Builder, e *mysqlEngine, opts metaengine.ExplainAggregateOptions) {
	if opts.Distinct != "" {
		fmt.Fprintf(b,
			`SELECT DISTINCT %s AS dv FROM meta_map WHERE collection = ?`,
			e.jsonFieldExpr(opts.Distinct))

		return
	}

	if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				e.aggExpr(s.Fn, s.Column),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		cols := strings.Join(selectCols, ", ")
		if opts.GroupBy != "" {
			fmt.Fprintf(b,
				`SELECT %s AS group_key, %s FROM meta_map WHERE collection = ?`,
				e.jsonFieldExpr(opts.GroupBy), cols)
		} else {
			fmt.Fprintf(b, `SELECT %s FROM meta_map WHERE collection = ?`, cols)
		}

		return
	}

	agg := e.aggExpr(opts.Fn, opts.Column)
	if opts.GroupBy != "" {
		fmt.Fprintf(
			b,
			`SELECT %s AS group_key, %s AS agg_val FROM meta_map WHERE collection = ?`,
			e.jsonFieldExpr(opts.GroupBy),
			agg,
		)
	} else {
		fmt.Fprintf(b, `SELECT %s FROM meta_map WHERE collection = ?`, agg)
	}
}
