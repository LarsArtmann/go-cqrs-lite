//go:build cgo

package duckdbengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan. It returns the SQL
// that PushdownMapScan would execute for the given collection and options,
// without running the query.
//
// For planned tables (collections with an applied LayoutPlan), the SQL uses
// direct column references. For standard tables, it uses json_extract with
// $N::json placeholders — the same SQL that PushdownMapScan builds internally.
func (e *duckdbEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) {
	if plan, ok := e.lookupPlan(collection); ok {
		return buildPlannedSelectQuery(plan, opts.Filters, opts.Sort, opts.Cursor, opts.Limit)
	}

	return explainStandardDuckDB(collection, opts.Filters, opts.Sort, opts.Cursor, opts.Limit)
}

// explainStandardDuckDB builds the SQL for a standard (non-planned) scan on
// meta_map using json_extract, mirroring the logic in PushdownMapScan.
func explainStandardDuckDB(
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{collection}

	b.WriteString(`SELECT value FROM meta_map WHERE collection = $1`)

	for _, f := range filters {
		path := jsonPath(f.Column)

		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok || len(values) == 0 {
				continue
			}

			placeholders := make([]string, len(values))
			for i, v := range values {
				jb, _ := json.Marshal(v)
				placeholders[i] = fmt.Sprintf("$%d::json", len(args)+1)
				args = append(args, string(jb))
			}

			fmt.Fprintf(&b, ` AND json_extract(value, '%s') IN (%s)`,
				path, strings.Join(placeholders, ", "))
		} else {
			jb, _ := json.Marshal(f.Value)
			fmt.Fprintf(&b, ` AND json_extract(value, '%s') %s $%d::json`,
				path, string(f.Op), len(args)+1)
			args = append(args, string(jb))
		}
	}

	if sort != nil && cursor != nil {
		path := jsonPath(sort.Column)
		op := ">"
		if sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(cursor)
		fmt.Fprintf(&b, ` AND json_extract(value, '%s') %s $%d::json`,
			path, op, len(args)+1)
		args = append(args, string(jb))
	}

	if sort != nil {
		path := jsonPath(sort.Column)
		fmt.Fprintf(&b, ` ORDER BY json_extract(value, '%s')`, path)
		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	return b.String(), args
}

// ExplainAggregateQuery implements metaengine.ExplainableAggregate. It returns
// the SQL that the aggregate methods (Aggregate, GroupedAggregate,
// MultiAggregate, MultiGroupedAggregate, DistinctValues) would execute,
// without running the query. Which SQL is built depends on ExplainAggregateOptions:
//   - Specs non-empty → multi/multi-grouped path
//   - Distinct non-empty → SELECT DISTINCT path
//   - GroupBy non-empty → GROUP BY path
//   - Otherwise → scalar aggregate path
func (e *duckdbEngine) ExplainAggregateQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainAggregateOptions,
) (string, []any) {
	plan, hasPlan := e.lookupPlan(collection)
	if !hasPlan {
		plan = metaengine.LayoutPlan{}
	}

	var b strings.Builder

	args := initialArgs(collection, plan)
	argIdx := initialArgIndex(plan)

	// Determine SELECT clause.
	if opts.Distinct != "" {
		fmt.Fprintf(&b, "SELECT DISTINCT %s AS dv %s",
			columnExpr(opts.Distinct, plan), fromClause(collection, plan))
	} else if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				aggExpr(s.Fn, s.Column, plan),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		if opts.GroupBy != "" {
			fmt.Fprintf(&b, "SELECT %s, %s %s",
				groupExpr(opts.GroupBy, plan)+" AS group_key",
				strings.Join(selectCols, ", "),
				fromClause(collection, plan))
		} else {
			fmt.Fprintf(&b, "SELECT %s %s",
				strings.Join(selectCols, ", "), fromClause(collection, plan))
		}
	} else {
		agg := aggExpr(opts.Fn, opts.Column, plan)
		if opts.GroupBy != "" {
			fmt.Fprintf(&b, "SELECT %s AS group_key, %s AS agg_val %s",
				groupExpr(opts.GroupBy, plan), agg, fromClause(collection, plan))
		} else {
			fmt.Fprintf(&b, "SELECT %s %s", agg, fromClause(collection, plan))
		}
	}

	// WHERE / AND filters. The planned path's FROM has no WHERE, so the
	// helper manages the WHERE/AND connector; the standard path's FROM
	// already emitted WHERE collection = $1 (nil = always AND).
	whereStarted := false

	var connector *bool
	if plan.Table != "" {
		connector = &whereStarted
	}

	for _, f := range opts.Filters {
		appendDuckDBFilter(&b, &args, &argIdx, connector, f, plan)
	}

	// GROUP BY.
	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

// Compile-time assertion.
var _ metaengine.ExplainableAggregate = (*duckdbEngine)(nil)
