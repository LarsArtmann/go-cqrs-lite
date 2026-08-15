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
// MariaDB gets JSON_EXTRACT/JSON_UNQUOTE forms (see dialect.go).
func (e *mysqlEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) { //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	var b strings.Builder

	args := []any{collection}

	b.WriteString(`SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ?`)

	for _, f := range opts.Filters {
		e.appendExplainFilter(&b, &args, f)
	}

	if opts.Sort != nil && opts.Cursor != nil {
		op := ">"
		if opts.Sort.Desc {
			op = "<"
		}

		fmt.Fprintf(&b, ` AND %s %s %s`,
			e.jsonCompareExpr(opts.Sort.Column), op, e.jsonParamPlaceholder())
		args = append(args, e.jsonFilterParam(opts.Cursor))
	}

	if opts.Sort != nil {
		fmt.Fprintf(&b, ` ORDER BY %s`, e.jsonFieldExpr(opts.Sort.Column))
		if opts.Sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if opts.Limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, opts.Limit+1)
	}

	return b.String(), args
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

	if opts.Distinct != "" {
		fmt.Fprintf(&b,
			`SELECT DISTINCT %s AS dv FROM meta_map WHERE collection = ?`,
			e.jsonFieldExpr(opts.Distinct))
	} else if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				e.aggExpr(s.Fn, s.Column),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		cols := strings.Join(selectCols, ", ")
		if opts.GroupBy != "" {
			fmt.Fprintf(&b,
				`SELECT %s AS group_key, %s FROM meta_map WHERE collection = ?`,
				e.jsonFieldExpr(opts.GroupBy), cols)
		} else {
			fmt.Fprintf(&b, `SELECT %s FROM meta_map WHERE collection = ?`, cols)
		}
	} else {
		agg := e.aggExpr(opts.Fn, opts.Column)
		if opts.GroupBy != "" {
			fmt.Fprintf(
				&b,
				`SELECT %s AS group_key, %s AS agg_val FROM meta_map WHERE collection = ?`,
				e.jsonFieldExpr(opts.GroupBy),
				agg,
			)
		} else {
			fmt.Fprintf(&b, `SELECT %s FROM meta_map WHERE collection = ?`, agg)
		}
	}

	for _, f := range opts.Filters {
		e.appendExplainFilter(&b, &args, f)
	}

	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

func (e *mysqlEngine) appendExplainFilter(
	b *strings.Builder,
	args *[]any,
	f metaengine.FilterSpec,
) {
	fmt.Fprintf(b, ` AND %s %s %s`,
		e.jsonCompareExpr(f.Column), string(f.Op), e.jsonParamPlaceholder())
	*args = append(*args, e.jsonFilterParam(f.Value))
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
		return fmt.Sprintf("SUM(%s)", field)
	}
}

// Compile-time assertions.
var (
	_ metaengine.ExplainableScan      = (*mysqlEngine)(nil)
	_ metaengine.ExplainableAggregate = (*mysqlEngine)(nil)
)
