package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
)

// plannedSQLiteEngine extends sqliteEngine with layout-planned tables.
// Instead of storing all data in meta_map with json_extract() scans,
// it creates dedicated tables with extracted columns and indexes.
//
// This is the Level 2 optimization: within-engine layout planning.
// The planned table enables the SQLite query planner to use B-tree indexes
// for WHERE/ORDER BY, achieving true O(logN + k) instead of O(N) for
// filtered scans.
type plannedSQLiteEngine struct {
	*sqliteEngine
	plans map[string]LayoutPlan // collection → layout plan
}

// NewPlannedSQLiteEngine creates a SQLite engine with layout-planned tables
// for the given collections. Collections without a plan fall back to the
// standard meta_map table.
func NewPlannedSQLiteEngine(database *sql.DB, plans []LayoutPlan) (Engine, error) {
	base, err := NewSQLiteEngine(database)
	if err != nil {
		return nil, err
	}

	eng := &plannedSQLiteEngine{
		sqliteEngine: base.(*sqliteEngine),
		plans:        make(map[string]LayoutPlan),
	}

	for _, plan := range plans {
		// Create the planned table + indexes.
		if _, err := database.ExecContext(context.Background(), plan.DDL()); err != nil {
			return nil, fmt.Errorf("planned engine: create table %s: %w", plan.Table, err)
		}

		eng.plans[plan.Collection] = plan
	}

	return eng, nil
}

// MapSet overrides the base implementation to populate extracted columns.
func (e *plannedSQLiteEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	plan, hasPlan := e.plans[col]
	if !hasPlan {
		return e.sqliteEngine.MapSet(ctx, col, key, value)
	}

	// Extract field values from the JSON value.
	valueJSON := encodeValue(value)
	extracted := extractFields(value, plan.Columns)

	// Build INSERT with extracted columns.
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

// MapGet overrides to read from the planned table.
func (e *plannedSQLiteEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	plan, hasPlan := e.plans[col]
	if !hasPlan {
		return e.sqliteEngine.MapGet(ctx, col, key)
	}

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

	var val any

	if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
		val = valStr
	}

	return val, true, nil
}

// MapDelete overrides to delete from the planned table.
func (e *plannedSQLiteEngine) MapDelete(ctx context.Context, col string, key any) error {
	plan, hasPlan := e.plans[col]
	if !hasPlan {
		return e.sqliteEngine.MapDelete(ctx, col, key)
	}

	_, err := e.db.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE key = ?", plan.Table),
		encodeKey(key))

	return err //nolint:wrapcheck // passthrough
}

// PushdownMapScan overrides to use indexed columns instead of json_extract.
func (e *plannedSQLiteEngine) PushdownMapScan(
	ctx context.Context,
	col string,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([]any, error) {
	plan, hasPlan := e.plans[col]
	if !hasPlan {
		return e.sqliteEngine.PushdownMapScan(ctx, col, filters, sort, cursor, limit)
	}

	var b strings.Builder

	args := []any{}

	b.WriteString(fmt.Sprintf("SELECT value FROM %s", plan.Table))

	// Push filters using direct column references (not json_extract).
	for i, f := range filters {
		if i == 0 {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		b.WriteString(fmt.Sprintf("%s %s ?", f.Column, string(f.Op)))
		args = append(args, f.Value)
	}

	// Push cursor using direct column reference.
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

		b.WriteString(fmt.Sprintf("%s %s ?", sort.Column, op))
		args = append(args, cursor)
	}

	// Push sort using direct column reference.
	if sort != nil {
		b.WriteString(fmt.Sprintf(" ORDER BY %s", sort.Column))
		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	// Push limit.
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

	// Try map[string]any first (JSON-decoded values).
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

	// Fallback: JSON round-trip to map[string]any.
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

// Compile-time assertion.
var _ Engine = (*plannedSQLiteEngine)(nil)
