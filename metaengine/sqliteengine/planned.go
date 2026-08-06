package sqliteengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// NewPlannedSQLiteEngine creates a SQLite engine with layout-planned tables
// for the given collections. Collections without a plan fall back to the
// standard meta_map table.
//
// As of ADR-0073 update, metaengine.Plan() auto-applies layouts for queries using
// FilterOnField/SortOnField — manual NewPlannedSQLiteEngine is only needed
// when you want explicit control over the metaengine.LayoutPlan (e.g., custom column types
// via metaengine.BuildLayoutPlanFromType[R]).
func NewPlannedSQLiteEngine(database *sql.DB, plans []metaengine.LayoutPlan) (metaengine.Engine, error) {
	eng, err := NewSQLiteEngine(database)
	if err != nil {
		return nil, err
	}

	sqlEng := eng.(*sqliteEngine)

	for _, plan := range plans {
		if err := sqlEng.registerLayout(plan); err != nil {
			return nil, fmt.Errorf("planned engine: create table %s: %w", plan.Table, err)
		}
	}

	return sqlEng, nil
}

// ApplyLayout implements metaengine.LayoutPlanner. It auto-generates a metaengine.LayoutPlan from
// the declared filter/sort field names and registers it on this engine. Called
// automatically by metaengine.Plan() for queries that use FilterOnField/SortOnField.
func (e *sqliteEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	if existing, exists := e.plans[collection]; exists {
		// Idempotent: same column set → no-op. Different columns → conflict.
		newPlan := metaengine.BuildLayoutPlan(collection, filterFields, sortFields)
		if !metaengine.PlansColumnCompatible(existing, newPlan) {
			return fmt.Errorf("%w: collection %q already has columns %v, requested %v",
				metaengine.ErrLayoutConflict, collection, existing.ColumnNames(), newPlan.ColumnNames())
		}

		return nil // already planned with same columns
	}

	plan := metaengine.BuildLayoutPlan(collection, filterFields, sortFields)

	if err := e.registerLayout(plan); err != nil {
		return fmt.Errorf("apply layout %q: %w", collection, err)
	}

	return nil
}


// registerLayout creates the planned table + indexes and stores the plan.
func (e *sqliteEngine) registerLayout(plan metaengine.LayoutPlan) error {
	if _, err := e.db.ExecContext(context.Background(), plan.DDL()); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// --- Planned table helpers (used when a collection has a metaengine.LayoutPlan) ---

func (e *sqliteEngine) mapSetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	return execPlannedSet(ctx, e.xd(), plan, key, value)
}

// execPlannedSet writes a key-value pair to a planned table with extracted columns.
// Works with both *sql.DB and *sql.Tx (for transactional MapUpdate).
type execContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func execPlannedSet(
	ctx context.Context,
	exec execContext,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	valueJSON := encodeValue(value)
	extracted := metaengine.ExtractFields(value, plan.Columns)

	quotedColNames := make([]string, 0, 2+len(plan.Columns))
	quotedColNames = append(quotedColNames, metaengine.QuoteIdent("key"), metaengine.QuoteIdent("value"))
	args := make([]any, 0, 2+len(plan.Columns))
	args = append(args, encodeKey(key), valueJSON)

	for _, c := range plan.Columns {
		quotedColNames = append(quotedColNames, metaengine.QuoteIdent(c.Name))
		args = append(args, extracted[c.Name])
	}

	placeholder := strings.Repeat("?,", len(quotedColNames))
	placeholder = "(" + placeholder[:len(placeholder)-1] + ")"

	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s) VALUES %s",
		metaengine.QuoteIdent(plan.Table), strings.Join(quotedColNames, ", "), placeholder,
	)

	_, err := exec.ExecContext(ctx, query, args...)

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) mapGetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) (any, bool, error) {
	var valStr string

	err := e.xd().QueryRowContext(ctx,
		fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
		encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return decodeJSONValue(valStr), true, nil
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
	// Inside outer tx: reuse it (SQLite doesn't support nested BEGIN).
	if e.txExec() != nil {
		xd := e.xd()

		var valStr string

		err := xd.QueryRowContext(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
			encodeKey(key)).Scan(&valStr)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err //nolint:wrapcheck // passthrough
		}

		var prev any
		if err == nil {
			prev = decodeJSONValue(valStr)
		}

		newVal := update(prev)

		return execPlannedSet(ctx, xd, plan, key, newVal)
	}

	return runTxReadModifyWrite(
		ctx, e.db, update,
		func(ctx context.Context, tx *sql.Tx) (any, error) {
			var valStr string
			if err := tx.QueryRowContext(ctx,
				fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
				encodeKey(key)).Scan(&valStr); err != nil {
				return nil, err //nolint:wrapcheck // ErrNoRows handled by caller
			}

			return decodeJSONValue(valStr), nil
		},
		func(ctx context.Context, tx *sql.Tx, newVal any) error {
			return execPlannedSet(ctx, tx, plan, key, newVal)
		},
	)
}

// runTxReadModifyWrite wraps a read-modify-write cycle in a single transaction
// so concurrent updates on the same key cannot interleave. The readFn may
// return sql.ErrNoRows (treated as nil prev); all other errors propagate.
func runTxReadModifyWrite(
	ctx context.Context,
	db *sql.DB,
	update func(prev any) any,
	readFn func(ctx context.Context, tx *sql.Tx) (any, error),
	writeFn func(ctx context.Context, tx *sql.Tx, newVal any) error,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = tx.Rollback() }()

	prev, readErr := readFn(ctx, tx)

	if readErr != nil && !errors.Is(readErr, sql.ErrNoRows) {
		return readErr
	}

	newVal := update(prev)

	if err := writeFn(ctx, tx, newVal); err != nil {
		return err
	}

	return tx.Commit() //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	query, args := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)

	rows, err := scanJSONValues(ctx, e.xd(), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}


