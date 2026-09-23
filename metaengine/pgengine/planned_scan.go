package pgengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Planned-table pushdown (D3 slice 1): PushdownMapScan over a collection
// with a registered LayoutPlan reads the extracted-column table instead of
// meta_map — native typed columns, native indexes, no JSONB path extraction.
// Mirrors metaengine/sqliteengine (raw_reader.go buildPlannedSelectQuery);
// the PG dialect uses $N placeholders and value::text for the JSONB value.
//
// Filter/sort/cursor values are validated against the declared column types
// at build time: a contradiction fails with
// metaengine.ErrPlannedColumnTypeMismatch (Rejection) before any SQL runs.

// validatePlannedFilterValue checks one value against the plan's declared
// column type, returning a classified Rejection on contradiction.
func validatePlannedFilterValue(
	plan metaengine.LayoutPlan,
	column string,
	val any,
) error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	typ, ok := metaengine.PlannedColumnType(plan, column)
	if !ok {
		return nil
	}

	if !metaengine.PlannedColumnTypeCompatible(typ, val) {
		return fmt.Errorf(
			"pgengine: %w: column %q is %s, got %T",
			metaengine.ErrPlannedColumnTypeMismatch, column, typ, val,
		)
	}

	return nil
}

// appendPGPlannedFilter writes one filter clause with $N placeholders and
// QuoteIdent-quoted column names; the clause mechanics live in
// metaengine.AppendPlannedFilter.
func appendPGPlannedFilter(
	b *strings.Builder,
	args *[]any,
	f metaengine.FilterSpec,
	started *bool,
) {
	metaengine.AppendPlannedFilter(
		b,
		args,
		f,
		started,
		metaengine.QuoteIdent,
		metaengine.DollarPlaceholders,
		", ",
	)
}

// buildPGPlannedScanQuery renders the planned-table pushdown SELECT: native
// column filters, ORDER BY on the declared column, keyset cursor predicate,
// and LIMIT n+1 for has-more detection.
func buildPGPlannedScanQuery(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any, error) {
	//art-dupl:accept cross-module SQL builder pattern — dep-isolated go.mod modules
	for _, f := range filters {
		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok {
				continue
			}

			for _, v := range values {
				if err := validatePlannedFilterValue(plan, f.Column, v); err != nil {
					return "", nil, err
				}
			}

			continue
		}

		if err := validatePlannedFilterValue(plan, f.Column, f.Value); err != nil {
			return "", nil, err
		}
	}

	if sort != nil && cursor != nil {
		if err := validatePlannedFilterValue(plan, sort.Column, cursor); err != nil {
			return "", nil, err
		}
	}

	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value::text FROM %s", metaengine.QuoteIdent(plan.Table))

	started := false

	for _, f := range filters {
		appendPGPlannedFilter(&b, &args, f, &started)
	}

	if sort != nil && cursor != nil {
		op := ">"
		if sort.Desc {
			op = "<"
		}

		if !started {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(&b, "%s %s $%d",
			metaengine.QuoteIdent(sort.Column), op, len(args)+1)

		args = append(args, cursor)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", metaengine.QuoteIdent(sort.Column))

		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, " LIMIT %d", limit+1)
	}

	return b.String(), args, nil
}

// pushdownMapScanPlanned executes the planned-table pushdown scan.
//
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *pgEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	query, args, err := buildPGPlannedScanQuery(plan, filters, sort, cursor, limit)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	rows, err := scanPGJSONValues(ctx, e.conn(ctx), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}
