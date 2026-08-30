package mysqlengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Planned-table pushdown (D3 slice 1): PushdownMapScan over a collection
// with a registered LayoutPlan reads the extracted-column table instead of
// meta_map — native typed columns, native indexes, no JSON_EXTRACT.
// Mirrors metaengine/sqliteengine and metaengine/pgengine; the MySQL/MariaDB
// dialect uses backtick identifiers and ? placeholders. Filter/sort/cursor
// values are validated against the declared column types at build time and
// fail with metaengine.ErrPlannedColumnTypeMismatch (Rejection) before any
// SQL runs.

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
			"mysqlengine: %w: column %q is %s, got %T",
			metaengine.ErrPlannedColumnTypeMismatch, column, typ, val,
		)
	}

	return nil
}

// appendPlannedFilter writes one filter clause with ? placeholders. The
// caller passes started so the first clause gets " WHERE " and the rest
// " AND ".
//
// art-dupl:accept cross-module SQL builder pattern — separate go.mod
func appendPlannedFilter(
	b *strings.Builder,
	args *[]any,
	f metaengine.FilterSpec,
	started *bool,
) {
	if f.Op == metaengine.FilterIn {
		values, ok := f.Value.([]any)
		if !ok || len(values) == 0 {
			return
		}

		if !*started {
			b.WriteString(" WHERE ")

			*started = true
		} else {
			b.WriteString(" AND ")
		}

		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"

			*args = append(*args, values[i])
		}

		fmt.Fprintf(b, "%s IN (%s)",
			backtickIdent(f.Column), strings.Join(placeholders, ", "))

		return
	}

	if !*started {
		b.WriteString(" WHERE ")

		*started = true
	} else {
		b.WriteString(" AND ")
	}

	fmt.Fprintf(b, "%s %s ?", backtickIdent(f.Column), string(f.Op))

	*args = append(*args, f.Value)
}

// buildPlannedScanQuery renders the planned-table pushdown SELECT: native
// column filters, ORDER BY on the declared column (native numeric ordering —
// the meta_map gcn_ twin-column dance is not needed on extracted columns),
// keyset cursor predicate, and LIMIT n+1 for has-more detection.
func buildPlannedScanQuery(
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

	fmt.Fprintf(&b, "SELECT CAST(value AS CHAR) FROM %s", backtickIdent(plan.Table))

	started := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &started)
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

		fmt.Fprintf(&b, "%s %s ?", backtickIdent(sort.Column), op)

		args = append(args, cursor)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", backtickIdent(sort.Column))

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
func (e *mysqlEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	query, args, err := buildPlannedScanQuery(plan, filters, sort, cursor, limit)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	rows, err := scanMySQLJSONValues(ctx, e.conn(), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	//art-dupl:accept cross-module SQL engine pattern — separate go.mod
	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}
