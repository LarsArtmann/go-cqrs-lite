package mysqlengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan. It returns the SQL
// that PushdownMapScan would execute for the given collection and options,
// without running the query. MySQL uses JSON path operators (value->'$.field')
// for column access and CAST(? AS JSON) placeholders for filter values.
func (e *mysqlEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) { //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	var b strings.Builder

	args := []any{collection}

	b.WriteString(`SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ?`)

	for _, f := range opts.Filters {
		appendMySQLExplainFilter(&b, &args, f)
	}

	if opts.Sort != nil && opts.Cursor != nil {
		op := ">"
		if opts.Sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(opts.Cursor)
		fmt.Fprintf(&b, ` AND value->'$.%s' %s CAST(? AS JSON)`,
			escapeJSONPath(opts.Sort.Column), op)
		args = append(args, string(jb))
	}

	if opts.Sort != nil {
		fmt.Fprintf(&b, ` ORDER BY value->'$.%s'`, escapeJSONPath(opts.Sort.Column))
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
			`SELECT DISTINCT value->'$.%s' AS dv FROM meta_map WHERE collection = ?`,
			escapeJSONPath(opts.Distinct))
	} else if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				mysqlAggExpr(s.Fn, escapeJSONPath(s.Column)),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		cols := strings.Join(selectCols, ", ")
		if opts.GroupBy != "" {
			fmt.Fprintf(&b,
				`SELECT value->'$.%s' AS group_key, %s FROM meta_map WHERE collection = ?`,
				escapeJSONPath(opts.GroupBy), cols)
		} else {
			fmt.Fprintf(&b, `SELECT %s FROM meta_map WHERE collection = ?`, cols)
		}
	} else {
		agg := mysqlAggExpr(opts.Fn, escapeJSONPath(opts.Column))
		if opts.GroupBy != "" {
			fmt.Fprintf(
				&b,
				`SELECT value->'$.%s' AS group_key, %s AS agg_val FROM meta_map WHERE collection = ?`,
				escapeJSONPath(opts.GroupBy),
				agg,
			)
		} else {
			fmt.Fprintf(&b, `SELECT %s FROM meta_map WHERE collection = ?`, agg)
		}
	}

	for _, f := range opts.Filters {
		appendMySQLExplainFilter(&b, &args, f)
	}

	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

func appendMySQLExplainFilter(
	b *strings.Builder,
	args *[]any,
	f metaengine.FilterSpec,
) {
	jb, _ := json.Marshal(f.Value)
	fmt.Fprintf(b, ` AND value->'$.%s' %s CAST(? AS JSON)`,
		escapeJSONPath(f.Column), string(f.Op))
	*args = append(*args, string(jb))
}

func mysqlAggExpr(fn metaengine.AggregateFn, column string) string {
	switch fn {
	case metaengine.AggregateCount:
		return "COUNT(*)"
	case metaengine.AggregateSum:
		return fmt.Sprintf("SUM(value->'$.%s')", column)
	case metaengine.AggregateMin:
		return fmt.Sprintf("MIN(value->'$.%s')", column)
	case metaengine.AggregateMax:
		return fmt.Sprintf("MAX(value->'$.%s')", column)
	case metaengine.AggregateAvg:
		return fmt.Sprintf("AVG(value->'$.%s')", column)
	default:
		return fmt.Sprintf("SUM(value->'$.%s')", column)
	}
}

// Compile-time assertions.
var (
	_ metaengine.ExplainableScan      = (*mysqlEngine)(nil)
	_ metaengine.ExplainableAggregate = (*mysqlEngine)(nil)
)
