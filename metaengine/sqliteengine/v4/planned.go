package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- Planned table helpers (used when a collection has a metaengine.LayoutPlan) ---

func (e *sqliteEngine) mapSetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	// Inside an ambient transaction: buffer the row for a chunked
	// multi-row INSERT OR REPLACE at commit (MF1.3) instead of executing
	// one statement per row. Buffers flush at commit, at the chunk cap, and
	// before any read/delete of the same table (sequential semantics).
	if txe := e.txFromCtx(ctx); txe != nil {
		quoted, row, err := plannedSetArgs(plan, key, value)
		if err != nil {
			return err
		}

		txe.pending.add(plan.Table, quoted, row)

		if txe.pending.len(plan.Table) >= plannedBatchChunk {
			if err := txe.pending.flushTable(ctx, txe.tx, plan.Table); err != nil {
				return fmt.Errorf("sqliteengine: planned chunk flush %s: %w", plan.Table, err)
			}
		}

		return nil
	}

	return execPlannedSet(ctx, e.xd(ctx), plan, key, value)
}

// execPlannedSet writes a key-value pair to a planned table with extracted columns.
// Works with both *sql.DB and *sql.Tx (for transactional MapUpdate).
type execContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// plannedSetArgs builds the quoted column list and one row's argument slice
// for a planned-table upsert. Shared by the immediate single-row path
// (execPlannedSet) and the buffered multi-row path (mapSetPlanned).
func plannedSetArgs(
	plan metaengine.LayoutPlan,
	key any,
	value any,
) ([]string, []any, error) {
	valueJSON, err := encodeValue(value)
	if err != nil {
		return nil, nil, err
	}

	extracted := metaengine.ExtractFields(
		value,
		plan.Columns,
	) // art-dupl:accept planned-write parity with duckdbengine/layout_planner.go — engines stay dep-isolated

	quotedColNames := make([]string, 0, 2+len(plan.Columns))
	quotedColNames = append(
		quotedColNames,
		metaengine.QuoteIdent("key"),
		metaengine.QuoteIdent("value"),
	)
	args := make([]any, 0, 2+len(plan.Columns))
	args = append(args, encodeKey(key), valueJSON)

	for _, c := range plan.Columns {
		quotedColNames = append(quotedColNames, metaengine.QuoteIdent(c.Name))
		args = append(args, extracted[c.Name])
	}

	return quotedColNames, args, nil
}

// flushPlannedTable flushes a table's buffered multi-row writes before an
// operation that must observe them (get/scan/update/delete — the
// read-your-writes and delete-after-set ordering rules of the batch
// contract). No-op outside a transaction or with nothing buffered.
func (e *sqliteEngine) flushPlannedTable(ctx context.Context, table string) error {
	if txe := e.txFromCtx(ctx); txe != nil {
		return txe.pending.flushTable(ctx, txe.tx, table) //nolint:wrapcheck // caller wraps
	}

	return nil
}

func execPlannedSet(
	ctx context.Context,
	exec execContext,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	quotedColNames, args, err := plannedSetArgs(plan, key, value)
	if err != nil {
		return err
	}

	placeholder := strings.Repeat("?,", len(quotedColNames))
	placeholder = "(" + placeholder[:len(placeholder)-1] + ")"

	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s) VALUES %s",
		metaengine.QuoteIdent(plan.Table), strings.Join(quotedColNames, ", "), placeholder,
	)

	_, err = exec.ExecContext(ctx, query, args...)

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) mapGetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) (any, bool, error) {
	// Read-your-writes: buffered batch rows must be visible to a get that
	// follows them in the same transaction (the batch contract's
	// flush-before-read rule).
	if err := e.flushPlannedTable(ctx, plan.Table); err != nil {
		return nil, false, fmt.Errorf("sqliteengine: planned flush-before-get %s: %w", plan.Table, err)
	}

	var valStr string

	err := e.xd(ctx).QueryRowContext(ctx,
		fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
		encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return metaengine.DecodeStreamValue(valStr), true, nil
}

// mapUpdatePlanned performs an atomic read-modify-write on a planned table.
// Same transaction pattern as the standard MapUpdate but reads/writes the
// planned table (with extracted columns) instead of meta_map.
func (e *sqliteEngine) mapUpdatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	update func(prev any) any,
) error {
	// Read-your-writes: the RMW read must observe buffered batch rows.
	if err := e.flushPlannedTable(ctx, plan.Table); err != nil {
		return fmt.Errorf("sqliteengine: planned flush-before-update %s: %w", plan.Table, err)
	}

	// Inside outer tx: reuse it (SQLite doesn't support nested BEGIN).
	if e.txExec(ctx) != nil {
		xd := e.xd(ctx)

		var valStr string

		err := xd.QueryRowContext(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
			encodeKey(key)).Scan(&valStr)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err //nolint:wrapcheck // passthrough
		}

		var prev any
		if err == nil {
			prev = metaengine.DecodeStreamValue(valStr)
		}

		newVal := update(prev)

		return execPlannedSet(ctx, xd, plan, key, newVal)
	}

	return runTxReadModifyWrite(
		ctx, e.db, update,
		func(ctx context.Context, exec metaengine.SQLExec) (any, error) {
			var valStr string
			if err := exec.QueryRowContext(
				ctx,
				fmt.Sprintf(
					"SELECT value FROM %s WHERE key = ?",
					metaengine.QuoteIdent(plan.Table),
				),
				encodeKey(key),
			).Scan(&valStr); err != nil {
				return nil, err //nolint:wrapcheck // ErrNoRows handled by caller
			}

			return metaengine.DecodeStreamValue(valStr), nil
		},
		func(ctx context.Context, exec metaengine.SQLExec, newVal any) error {
			return execPlannedSet(ctx, exec, plan, key, newVal)
		},
	)
}

// runTxReadModifyWrite wraps a read-modify-write cycle in a single
// transaction so concurrent updates on the same key cannot interleave. The
// transaction is opened BEGIN IMMEDIATE (see runImmediateTx) so the
// read-then-write upgrade can never hit SQLITE_BUSY_SNAPSHOT. The readFn may
// return sql.ErrNoRows (treated as nil prev); all other errors propagate.
func runTxReadModifyWrite(
	ctx context.Context,
	db *sql.DB,
	update func(prev any) any,
	readFn func(ctx context.Context, exec metaengine.SQLExec) (any, error),
	writeFn func(ctx context.Context, exec metaengine.SQLExec, newVal any) error,
) error {
	return runImmediateTx(ctx, db, func(ctx context.Context, exec metaengine.SQLExec) error {
		prev, readErr := readFn(ctx, exec)

		if readErr != nil && !errors.Is(readErr, sql.ErrNoRows) {
			return readErr
		}

		newVal := update(prev)

		return writeFn(ctx, exec, newVal)
	})
}

func (e *sqliteEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	// Read-your-writes: scans must observe buffered batch rows.
	if err := e.flushPlannedTable(ctx, plan.Table); err != nil {
		return metaengine.ScanResult{}, fmt.Errorf(
			"sqliteengine: planned flush-before-scan %s: %w", plan.Table, err)
	}

	query, args, err := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	rows, keys, err := scanJSONValuesWithKeys(ctx, e.xd(ctx), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastDecodedRowCursor(
			rows[len(rows)-1],
			sort.Column,
			[]byte(keys[len(keys)-1]),
		)
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}
