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
