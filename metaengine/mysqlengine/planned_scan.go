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

// appendPlannedFilter writes one filter clause with ? placeholders and
// MySQL backtick-quoted column names; the clause mechanics live in
// metaengine.AppendPlannedFilter.
func appendPlannedFilter(
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
		backtickIdent,
		metaengine.QuestionPlaceholders,
		", ",
	)
}

// buildPlannedScanQuery renders the planned-table pushdown SELECT: native
// column filters, ORDER BY on the declared column (native numeric ordering —
// the meta_map gcn_ twin-column dance is not needed on extracted columns),
// keyset cursor predicate, and LIMIT n+1 for has-more detection. The SELECT
// carries the key column so keyset scans can tiebreak and emit compound
// cursors; cursor predicate and ORDER BY go through the shared metaengine
// keyset helpers (key tiebreak ALWAYS ascending, mirroring every other
// engine).
func buildPlannedScanQuery(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any, error) {
	//art-dupl:accept cross-module SQL builder pattern — dep-isolated go.mod modules
	if err := metaengine.ValidateFilterSpecs(filters); err != nil {
		return "", nil, err
	}

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

	// A compound cursor validates its SORT component against the column type;
	// the struct itself is not a column value.
	cursorVal := cursor
	if skc, ok := cursor.(metaengine.SortKeyCursor); ok {
		cursorVal = skc.Sort
	}

	if sort != nil && cursorVal != nil {
		if err := validatePlannedFilterValue(plan, sort.Column, cursorVal); err != nil {
			return "", nil, err
		}
	}

	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT CAST(value AS CHAR), %s FROM %s", backtickIdent("key"), backtickIdent(plan.Table))

	started := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &started)
	}

	if sort != nil && cursor != nil {
		metaengine.AppendKeysetCursorPredicate(
			&b, &args, &started,
			backtickIdent(sort.Column), backtickIdent("key"),
			sort.Desc, cursor,
			func(n int) string { return "?" },
		)
	}

	if sort != nil {
		metaengine.AppendKeysetOrder(&b, backtickIdent(sort.Column), backtickIdent("key"), sort.Desc)
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

	rows, keys, err := scanMySQLJSONValuesWithKeys(ctx, e.conn(ctx), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	//art-dupl:accept cross-module SQL engine pattern — separate go.mod
	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastDecodedRowCursor(rows[len(rows)-1], sort.Column, []byte(keys[len(keys)-1]))
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}
