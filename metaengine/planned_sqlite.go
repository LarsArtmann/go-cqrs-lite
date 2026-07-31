package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NewPlannedSQLiteEngine creates a SQLite engine with layout-planned tables
// for the given collections. Collections without a plan fall back to the
// standard meta_map table.
//
// As of ADR-0073 update, Plan() auto-applies layouts for queries using
// FilterOnField/SortOnField — manual NewPlannedSQLiteEngine is only needed
// when you want explicit control over the LayoutPlan (e.g., custom column types
// via BuildLayoutPlanFromType[R]).
func NewPlannedSQLiteEngine(database *sql.DB, plans []LayoutPlan) (Engine, error) {
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

// ApplyLayout implements LayoutPlanner. It auto-generates a LayoutPlan from
// the declared filter/sort field names and registers it on this engine. Called
// automatically by Plan() for queries that use FilterOnField/SortOnField.
func (e *sqliteEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	if e.plans == nil {
		e.plans = make(map[string]LayoutPlan)
	}

	if existing, exists := e.plans[collection]; exists {
		// Idempotent: same column set → no-op. Different columns → conflict.
		newPlan := BuildLayoutPlan(collection, filterFields, sortFields)
		if !plansColumnCompatible(existing, newPlan) {
			return fmt.Errorf("%w: collection %q already has columns %v, requested %v",
				ErrLayoutConflict, collection, existing.ColumnNames(), newPlan.ColumnNames())
		}

		return nil // already planned with same columns
	}

	plan := BuildLayoutPlan(collection, filterFields, sortFields)

	if err := e.registerLayout(plan); err != nil {
		return fmt.Errorf("apply layout %q: %w", collection, err)
	}

	return nil
}

// plansColumnCompatible returns true when two plans have the same set of
// column names (order-independent). Used to detect layout conflicts.
func plansColumnCompatible(a, b LayoutPlan) bool {
	ac := a.ColumnNames()

	bc := b.ColumnNames()
	if len(ac) != len(bc) {
		return false
	}

	bset := make(map[string]bool, len(bc))
	for _, c := range bc {
		bset[c] = true
	}

	for _, c := range ac {
		if !bset[c] {
			return false
		}
	}

	return true
}

// registerLayout creates the planned table + indexes and stores the plan.
func (e *sqliteEngine) registerLayout(plan LayoutPlan) error {
	if _, err := e.db.ExecContext(context.Background(), plan.DDL()); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	if e.plans == nil {
		e.plans = make(map[string]LayoutPlan)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// --- Planned table helpers (used when a collection has a LayoutPlan) ---

func (e *sqliteEngine) mapSetPlanned(
	ctx context.Context,
	plan LayoutPlan,
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
	plan LayoutPlan,
	key any,
	value any,
) error {
	valueJSON := encodeValue(value)
	extracted := extractFields(value, plan.Columns)

	colNames := []string{"key", "value"}
	args := []any{encodeKey(key), valueJSON}

	for _, c := range plan.Columns {
		colNames = append(colNames, c.Name)
		args = append(args, extracted[c.Name])
	}

	placeholder := strings.Repeat("?,", len(colNames))
	placeholder = "(" + placeholder[:len(placeholder)-1] + ")"

	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s) VALUES %s",
		plan.Table, strings.Join(colNames, ", "), placeholder,
	)

	_, err := exec.ExecContext(ctx, query, args...)

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) mapGetPlanned(
	ctx context.Context,
	plan LayoutPlan,
	key any,
) (any, bool, error) {
	var valStr string

	err := e.xd().QueryRowContext(ctx,
		fmt.Sprintf("SELECT value FROM %s WHERE key = ?", plan.Table),
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
	plan LayoutPlan,
	key any,
	update func(prev any) any,
) error {
	// Inside outer tx: reuse it (SQLite doesn't support nested BEGIN).
	if e.txExec() != nil {
		xd := e.xd()

		var valStr string

		err := xd.QueryRowContext(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", plan.Table),
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
				fmt.Sprintf("SELECT value FROM %s WHERE key = ?", plan.Table),
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
	plan LayoutPlan,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([]any, error) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", plan.Table)

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	if sort != nil && cursor != nil {
		if !whereStarted {
			b.WriteString(" WHERE ")

			whereStarted = true
		} else {
			b.WriteString(" AND ")
		}

		op := ">"
		if sort.Desc {
			op = "<"
		}

		fmt.Fprintf(&b, "%s %s ?", sort.Column, op)

		args = append(args, cursor)
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

	return scanJSONValues(ctx, e.xd(), b.String(), args...)
}

// extractFields pulls field values from a Go value (struct or map) for the
// planned columns. Missing fields produce nil (stored as NULL).
//
// Structs use a reflect fast path (no JSON marshal/unmarshal on writes).
// Maps and other types fall back to JSON round-trip.
func extractFields(value any, columns []PlannedColumn) map[string]any {
	result := make(map[string]any, len(columns))

	if m, ok := value.(map[string]any); ok {
		for _, c := range columns {
			for k, v := range m {
				if strings.EqualFold(k, c.Name) {
					result[c.Name] = v

					break
				}
			}
		}

		return result
	}

	// Reflect fast path for structs — avoids JSON marshal/unmarshal cycle.
	rv := reflect.ValueOf(value)

	if rv.IsValid() && rv.Kind() == reflect.Struct {
		rt := rv.Type()

		for _, c := range columns {
			for i := range rt.NumField() {
				f := rt.Field(i)
				if !f.IsExported() {
					continue
				}

				fieldName := jsonFieldName(f)

				if strings.EqualFold(fieldName, c.Name) {
					result[c.Name] = rv.Field(i).Interface()

					break
				}
			}
		}

		return result
	}

	// Fallback: JSON round-trip for non-struct values.
	b, err := json.Marshal(value)
	if err != nil {
		return result
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return result
	}

	for _, c := range columns {
		for k, v := range m {
			if strings.EqualFold(k, c.Name) {
				result[c.Name] = v

				break
			}
		}
	}

	return result
}

// jsonFieldName returns the JSON field name for a struct field, respecting
// json tags. Falls back to the Go field name when no tag is present.
func jsonFieldName(f reflect.StructField) string {
	if tag := f.Tag.Get("json"); tag != "" {
		if name, _, _ := strings.Cut(tag, ","); name != "" {
			return name
		}
	}

	return f.Name
}
