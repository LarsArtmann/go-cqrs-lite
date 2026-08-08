package pgengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan. It returns the SQL
// that PushdownMapScan would execute for the given collection and options,
// without running the query. Postgres uses JSONB operators (value->'field')
// for column access and $N::jsonb placeholders for filter values.
func (e *pgEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) { //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	var b strings.Builder

	args := []any{collection}

	b.WriteString(`SELECT value::text FROM meta_map WHERE collection = $1`)

	for _, f := range opts.Filters {
		appendPGFilter(&b, &args, f)
	}

	if opts.Sort != nil && opts.Cursor != nil {
		op := ">"
		if opts.Sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(opts.Cursor)
		fmt.Fprintf(&b, ` AND value->'%s' %s $%d::jsonb`,
			escapeJSONKey(opts.Sort.Column), op, len(args)+1)
		args = append(args, string(jb))
	}

	if opts.Sort != nil {
		fmt.Fprintf(&b, ` ORDER BY value->'%s'`, escapeJSONKey(opts.Sort.Column))
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
// the SQL that the aggregate methods would execute, without running the query.
// Postgres uses JSONB operators (value->'field') with ::float8 casts for
// aggregate functions and $N placeholders.
func (e *pgEngine) ExplainAggregateQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainAggregateOptions,
) (string, []any) {
	var b strings.Builder

	args := []any{collection}

	// Determine SELECT clause.
	if opts.Distinct != "" {
		fmt.Fprintf(&b, `SELECT DISTINCT value->'%s' AS dv FROM meta_map WHERE collection = $1`,
			escapeJSONKey(opts.Distinct))
	} else if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				pgAggExpr(s.Fn, s.Column),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		cols := strings.Join(selectCols, ", ")
		if opts.GroupBy != "" {
			fmt.Fprintf(&b, `SELECT value->>'%s' AS group_key, %s FROM meta_map WHERE collection = $1`,
				escapeJSONKey(opts.GroupBy), cols)
		} else {
			fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = $1", cols)
		}
	} else {
		agg := pgAggExpr(opts.Fn, opts.Column)
		if opts.GroupBy != "" {
			fmt.Fprintf(&b, `SELECT value->>'%s' AS group_key, %s AS agg_val FROM meta_map WHERE collection = $1`,
				escapeJSONKey(opts.GroupBy), agg)
		} else {
			fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = $1", agg)
		}
	}

	// WHERE / AND filters.
	for _, f := range opts.Filters {
		appendPGFilter(&b, &args, f)
	}

	// GROUP BY.
	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

// Compile-time assertions.
var (
	_ metaengine.ExplainableScan       = (*pgEngine)(nil)
	_ metaengine.ExplainableAggregate = (*pgEngine)(nil)
)
