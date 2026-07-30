package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
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

	if _, exists := e.plans[collection]; exists {
		return nil // already planned (idempotent)
	}

	plan := BuildLayoutPlan(collection, filterFields, sortFields)

	if err := e.registerLayout(plan); err != nil {
		return fmt.Errorf("apply layout %q: %w", collection, err)
	}

	return nil
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

func (e *sqliteEngine) mapSetPlanned(ctx context.Context, plan LayoutPlan, key any, value any) error {
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

	_, err := e.db.ExecContext(ctx, query, args...)

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) mapGetPlanned(ctx context.Context, plan LayoutPlan, key any) (any, bool, error) {
	var valStr string

	err := e.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT value FROM %s WHERE key = ?", plan.Table),
		encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return jsonValue(valStr), true, nil
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

	for i, f := range filters {
		if i == 0 {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(&b, "%s %s ?", f.Column, string(f.Op))
		args = append(args, f.Value)
	}

	if sort != nil && cursor != nil {
		if len(filters) == 0 {
			b.WriteString(" WHERE ")
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

	return scanJSONValues(ctx, e.db, b.String(), args...)
}

// extractFields pulls field values from a Go value (struct or map) for the
// planned columns. Missing fields produce nil (stored as NULL).
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
