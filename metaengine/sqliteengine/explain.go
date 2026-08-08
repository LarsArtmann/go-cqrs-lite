package sqliteengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan.
func (e *sqliteEngine) ExplainScanQuery(
	_ context.Context,
	col string,
	opts metaengine.ExplainOptions,
) (string, []any) {
	filters := opts.Filters

	if plan, ok := e.plans[col]; ok {
		return explainPlanned(plan, filters, opts.Sort, opts.Limit)
	}

	return explainStandard(col, filters, opts.Sort, opts.Limit)
}

func explainStandard(
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{col}

	b.WriteString(`SELECT value FROM meta_map WHERE collection = ?`)

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	if sort != nil {
		fmt.Fprintf(&b, ` ORDER BY json_extract(value, '%s')`, jsonPath(sort.Column))

		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		b.WriteString(` LIMIT ?`)

		args = append(args, limit+1)
	}

	return b.String(), args
}

func explainPlanned(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", metaengine.QuoteIdent(sort.Column))

		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		b.WriteString(" LIMIT ?")

		args = append(args, limit+1)
	}

	return b.String(), args
}

// ExplainAggregateQuery implements metaengine.ExplainableAggregate. It returns
// the SQL for aggregate queries without executing them.
func (e *sqliteEngine) ExplainAggregateQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainAggregateOptions,
) (string, []any) {
	planned := false

	var plan metaengine.LayoutPlan

	if p, ok := e.plans[collection]; ok {
		plan = p
		planned = true
	}

	var b strings.Builder

	args := []any{}

	// Determine SELECT + FROM clause.
	if !planned {
		args = append(args, collection)
	}

	if opts.Distinct != "" {
		if planned {
			fmt.Fprintf(&b, "SELECT DISTINCT %s AS dv FROM %s",
				metaengine.QuoteIdent(opts.Distinct), metaengine.QuoteIdent(plan.Table))
		} else {
			fmt.Fprintf(
				&b,
				`SELECT DISTINCT json_extract(value, '%s') AS dv FROM meta_map WHERE collection = ?`,
				jsonPath(opts.Distinct),
			)
		}
	} else if len(opts.Specs) > 0 {
		selectCols := make([]string, len(opts.Specs))
		for i, s := range opts.Specs {
			selectCols[i] = fmt.Sprintf("%s AS %s",
				aggExprSQLite(s.Fn, s.Column, planned),
				metaengine.QuoteIdent(s.AliasOr()))
		}

		cols := strings.Join(selectCols, ", ")
		if opts.GroupBy != "" {
			grpCol := aggGroupExprSQLite(opts.GroupBy, planned)
			if planned {
				fmt.Fprintf(&b, "SELECT %s AS group_key, %s FROM %s",
					grpCol, cols, metaengine.QuoteIdent(plan.Table))
			} else {
				fmt.Fprintf(&b, "SELECT %s AS group_key, %s FROM meta_map WHERE collection = ?",
					grpCol, cols)
			}
		} else {
			if planned {
				fmt.Fprintf(&b, "SELECT %s FROM %s", cols, metaengine.QuoteIdent(plan.Table))
			} else {
				fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = ?", cols)
			}
		}
	} else {
		agg := aggExprSQLite(opts.Fn, opts.Column, planned)
		if opts.GroupBy != "" {
			grpCol := aggGroupExprSQLite(opts.GroupBy, planned)
			if planned {
				fmt.Fprintf(&b, "SELECT %s AS group_key, %s AS agg_val FROM %s",
					grpCol, agg, metaengine.QuoteIdent(plan.Table))
			} else {
				fmt.Fprintf(
					&b,
					"SELECT %s AS group_key, %s AS agg_val FROM meta_map WHERE collection = ?",
					grpCol,
					agg,
				)
			}
		} else {
			if planned {
				fmt.Fprintf(&b, "SELECT %s FROM %s", agg, metaengine.QuoteIdent(plan.Table))
			} else {
				fmt.Fprintf(&b, "SELECT %s FROM meta_map WHERE collection = ?", agg)
			}
		}
	}

	// WHERE / AND filters.
	whereStarted := false

	for _, f := range opts.Filters {
		if planned {
			appendPlannedFilter(&b, &args, f, &whereStarted)
		} else {
			appendStandardFilter(&b, &args, f)
		}
	}

	// GROUP BY.
	if opts.GroupBy != "" {
		b.WriteString(" GROUP BY group_key")
	}

	return b.String(), args
}

// aggGroupExprSQLite builds the GROUP BY column expression for SQLite.
func aggGroupExprSQLite(column string, planned bool) string {
	if planned {
		return metaengine.QuoteIdent(column)
	}

	return fmt.Sprintf("json_extract(value, '%s')", jsonPath(column))
}

// Compile-time assertion.
var _ metaengine.ExplainableAggregate = (*sqliteEngine)(nil)
