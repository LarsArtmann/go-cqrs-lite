package metaengine

import (
	"context"
	"fmt"
	"strings"
)

// ExplainOptions controls what EXPLAIN returns.
type ExplainOptions struct {
	Filters []FilterSpec
	Sort    *SortSpec
	Limit   int
}

// Explain returns the SQL that would execute for a scan query, without running
// it. Useful for debugging pushdown, verifying index usage, and understanding
// query plans.
//
//	reader := metaengine.NewReader[V](store, "find_user")
//	sql, args := reader.Explain(ctx,
//	    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
//	)
func (r *TypedReader[V]) Explain(
	ctx context.Context,
	opts ...ScanOption,
) (query string, args []any) {
	cfg := scanConfig{limit: 100}

	for _, opt := range opts {
		opt(&cfg)
	}

	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return "-- no engine for collection " + r.collection, nil
	}

	if se, ok := eng.(interface {
		explainScan(ctx context.Context, col string, cfg scanConfig) (string, []any)
	}); ok {
		return se.explainScan(ctx, r.collection, cfg)
	}

	return "-- EXPLAIN not supported by engine " + eng.Profile().Name, nil
}

// explainScan generates the SQL that would execute for a scan on sqliteEngine.
func (e *sqliteEngine) explainScan(
	ctx context.Context,
	col string,
	cfg scanConfig,
) (string, []any) {
	filters := cfg.filters

	// Expand ranges
	for _, rg := range cfg.ranges {
		filters = append(
			filters,
			FilterSpec{Column: rg.Column, Op: FilterGe, Value: rg.Low},
			FilterSpec{Column: rg.Column, Op: FilterLe, Value: rg.High},
		)
	}

	if plan, ok := e.plans[col]; ok {
		return explainPlanned(plan, filters, cfg.sort, cfg.limit)
	}

	return explainStandard(col, filters, cfg.sort, cfg.limit)
}

func explainStandard(col string, filters []FilterSpec, sort *SortSpec, limit int) (string, []any) {
	var b strings.Builder

	args := []any{col}

	b.WriteString(`SELECT value FROM meta_map WHERE collection = ?`)

	for _, f := range filters {
		b.WriteString(fmt.Sprintf(` AND json_extract(value, '$.%s') %s ?`, f.Column, string(f.Op)))
		args = append(args, f.Value)
	}

	if sort != nil {
		b.WriteString(fmt.Sprintf(` ORDER BY json_extract(value, '$.%s')`, sort.Column))
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
	plan LayoutPlan,
	filters []FilterSpec,
	sort *SortSpec,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", plan.Table)

	for i, f := range filters {
		if i == 0 {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(&b, "%s %s ?", f.Column, string(f.Op))
		args = append(args, f.Value)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", sort.Column)
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
