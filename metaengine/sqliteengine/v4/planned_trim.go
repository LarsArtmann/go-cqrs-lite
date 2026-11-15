package sqliteengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TrimPlannedBefore implements metaengine.PlannedTableSweeper: bulk by-time
// deletion of a planned table's rows by an extracted INTEGER column (JSON
// field name), one transaction, returning the rows removed.
//
// Column resolution is engine-side by design: the engine owns its layout
// plans, so the caller addresses the sweep by collection + JSON field name —
// never by raw table/column identifiers. A field that is not an extracted
// column, or whose column type is not INTEGER, is REJECTED (a TEXT-typed
// time column compared numerically would trim arbitrary rows — the
// name-based BuildLayoutPlan types every column TEXT, so this guard is what
// makes the sweep safe).
//
// Delete-after-set ordering (the MapDelete batch contract): buffered planned
// writes for the table flush BEFORE the sweep executes, or deferred INSERTs
// would resurrect trimmed rows after the sweep committed.
func (e *sqliteEngine) TrimPlannedBefore(ctx context.Context, collection, field string, cutoff int64) (int64, error) {
	plan, ok := e.plans[collection]
	if !ok {
		return 0, fmt.Errorf(
			"sqliteengine: planned sweep %s: %w: no layout plan for collection (not a planned table?)",
			collection, metaengine.ErrNoQueryForName,
		)
	}

	column, err := plannedIntegerColumn(plan, field)
	if err != nil {
		return 0, fmt.Errorf("sqliteengine: planned sweep %s.%s: %w", collection, field, err)
	}

	if err := e.flushPlannedTable(ctx, plan.Table); err != nil {
		return 0, fmt.Errorf("sqliteengine: planned flush-before-sweep %s: %w", plan.Table, err)
	}

	var removed int64

	err = e.RunInTx(ctx, func(ctx context.Context) error {
		res, execErr := e.xd(ctx).ExecContext(ctx, fmt.Sprintf(
			"DELETE FROM %s WHERE %s < ?",
			metaengine.QuoteIdent(plan.Table),
			metaengine.QuoteIdent(column),
		), cutoff)
		if execErr != nil {
			return fmt.Errorf("sqliteengine: planned sweep %s.%s: %w", plan.Table, column, execErr)
		}

		removed, execErr = res.RowsAffected()
		if execErr != nil {
			return fmt.Errorf("sqliteengine: planned sweep %s.%s: rows affected: %w", plan.Table, column, execErr)
		}

		return nil
	})
	if err != nil {
		return removed, err //nolint:wrapcheck // RunInTx errors already carry full context
	}

	return removed, nil
}

// plannedIntegerColumn resolves a JSON field name to its extracted column
// and refuses non-INTEGER columns (the numeric cutoff comparison is only
// honest on genuinely numeric columns).
func plannedIntegerColumn(plan metaengine.LayoutPlan, field string) (string, error) {
	for _, col := range plan.Columns {
		if col.Name != field {
			continue
		}

		if !strings.EqualFold(col.Type, "INTEGER") {
			return "", fmt.Errorf(
				"column %q has type %s (want INTEGER: numeric cutoff comparison is only honest on integer columns)",
				field, col.Type,
			)
		}

		return col.Name, nil
	}

	return "", fmt.Errorf(
		"no extracted column for field %q (declare it via FilterOnField/SortOnField so it lands in the layout plan)",
		field,
	)
}
