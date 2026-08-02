package duckdbengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ApplyLayout implements metaengine.LayoutPlanner. It creates a dedicated
// planned table with extracted columns and ART indexes for the declared
// filter/sort fields. DuckDB does not support expression indexes on JSON
// paths, so the planned-table approach (same as SQLite) is used: each
// collection with a layout gets its own table with typed columns that
// DuckDB's zone maps can prune.
//
// After ApplyLayout, MapSet writes to the planned table (with extracted
// columns) instead of meta_map, and PushdownMapScan queries the planned
// table with direct column references instead of json_extract.
func (e *duckdbEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	plan := metaengine.BuildLayoutPlan(collection, filterFields, sortFields)

	return e.ApplyLayoutPlan(plan)
}

// ApplyLayoutPlan implements metaengine.LayoutPlanApplier. It creates the
// planned table from a fully-built LayoutPlan with reflection-derived column
// types. This is the path used by WithColumnarLayout, where every exported
// field of the result type becomes a native column.
func (e *duckdbEngine) ApplyLayoutPlan(plan metaengine.LayoutPlan) error {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	if existing, exists := e.plans[plan.Collection]; exists {
		if !plansColumnCompatible(existing, plan) {
			return fmt.Errorf(
				"%w: collection %q already has columns %v, requested %v",
				metaengine.ErrLayoutConflict,
				plan.Collection,
				existing.ColumnNames(),
				plan.ColumnNames(),
			)
		}

		return nil
	}

	if _, err := e.db.ExecContext(context.Background(), plan.DDL()); err != nil {
		return fmt.Errorf("duckdbengine.ApplyLayoutPlan: create table %s: %w", plan.Table, err)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// --- Planned table helpers (used when a collection has a LayoutPlan) ---

func (e *duckdbEngine) mapSetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("duckdbengine.mapSetPlanned: marshal: %w", err)
	}

	extracted := extractFields(value, plan.Columns)

	quotedCols := make([]string, 0, 2+len(plan.Columns))
	quotedCols = append(quotedCols, quoteIdent("key"), quoteIdent("value"))

	placeholders := make([]string, 0, 2+len(plan.Columns))
	placeholders = append(placeholders, "$1", "$2")

	args := make([]any, 0, 2+len(plan.Columns))
	args = append(args, fmt.Sprint(key), string(data))

	for i, c := range plan.Columns {
		quotedCols = append(quotedCols, quoteIdent(c.Name))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+3))
		args = append(args, coerceForColumn(extracted[c.Name], c))
	}

	setCols := make([]string, 0, 1+len(plan.Columns))
	setCols = append(
		setCols,
		fmt.Sprintf("%s = excluded.%s", quoteIdent("value"), quoteIdent("value")),
	)

	for _, c := range plan.Columns {
		setCols = append(
			setCols,
			fmt.Sprintf("%s = excluded.%s", quoteIdent(c.Name), quoteIdent(c.Name)),
		)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quoteIdent(plan.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
		quoteIdent("key"),
		strings.Join(setCols, ", "),
	)

	if _, err := e.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("duckdbengine.mapSetPlanned: %w", err)
	}

	return nil
}

func (e *duckdbEngine) mapGetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) (any, bool, error) {
	var raw string

	err := e.db.QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT value FROM %s WHERE key = $1", quoteIdent(plan.Table)),
		fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("duckdbengine.mapGetPlanned: %w", err)
	}

	var val any
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return nil, false, fmt.Errorf("duckdbengine.mapGetPlanned: unmarshal: %w", err)
	}

	return val, true, nil
}

func (e *duckdbEngine) mapDeletePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) error {
	_, err := e.db.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE key = $1", quoteIdent(plan.Table)),
		fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("duckdbengine.mapDeletePlanned: %w", err)
	}

	return nil
}

func (e *duckdbEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	query, args := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)

	rows, err := scanDuckDBJSONValues(ctx, e.db, query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}

// buildPlannedSelectQuery constructs a parameterised SELECT for a planned
// table using DuckDB's $N placeholder syntax. Filters use direct column
// references (no json_extract), enabling DuckDB's zone maps and ART indexes
// to prune data blocks.
func buildPlannedSelectQuery(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", quoteIdent(plan.Table))

	whereStarted := false
	argIdx := 1

	for _, f := range filters {
		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok || len(values) == 0 {
				continue
			}

			writeWhereOrAnd(&b, &whereStarted)

			placeholders := make([]string, len(values))
			for i, v := range values {
				placeholders[i] = fmt.Sprintf("$%d", argIdx)
				args = append(args, v)
				argIdx++
			}

			fmt.Fprintf(&b, "%s IN (%s)", quoteIdent(f.Column), strings.Join(placeholders, ", "))
		} else {
			writeWhereOrAnd(&b, &whereStarted)

			fmt.Fprintf(&b, "%s %s $%d", quoteIdent(f.Column), string(f.Op), argIdx)
			args = append(args, f.Value)
			argIdx++
		}
	}

	if sort != nil && cursor != nil {
		writeWhereOrAnd(&b, &whereStarted)

		op := ">"
		if sort.Desc {
			op = "<"
		}

		fmt.Fprintf(&b, "%s %s $%d", quoteIdent(sort.Column), op, argIdx)
		args = append(args, cursor)
		argIdx++
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", quoteIdent(sort.Column))
		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, " LIMIT $%d", argIdx)
		args = append(args, limit+1)
	}

	return b.String(), args
}

// writeWhereOrAnd appends " WHERE " on the first call and " AND " on later
// calls, updating the flag so subsequent calls use AND.
func writeWhereOrAnd(b *strings.Builder, whereStarted *bool) {
	if !*whereStarted {
		b.WriteString(" WHERE ")
		*whereStarted = true

		return
	}

	b.WriteString(" AND ")
}

// extractFields pulls field values from a Go value for the planned columns.
// Missing fields produce nil (stored as NULL). Mirrors the metaengine
// extractFields logic (which is unexported).
func extractFields(value any, columns []metaengine.PlannedColumn) map[string]any {
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

// plansColumnCompatible returns true when two plans have the same set of
// column names (order-independent). Used to detect layout conflicts.
func plansColumnCompatible(a, b metaengine.LayoutPlan) bool {
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

// coerceForColumn converts a Go value to a type compatible with the planned
// column's SQL type. JSON decoding turns all numbers into float64, so this
// maps them back to INTEGER/REAL when the plan declares those types.
func coerceForColumn(value any, column metaengine.PlannedColumn) any {
	if value == nil {
		return nil
	}

	switch strings.ToUpper(column.Type) {
	case "INTEGER":
		return coerceInteger(value)
	case "REAL":
		return coerceReal(value)
	case "TEXT":
		return fmt.Sprint(value)
	default:
		return value
	}
}

// coerceInteger maps a Go value to int64. It handles bool (0/1), numeric
// strings, and all signed/unsigned integer and float widths via reflection.
func coerceInteger(value any) any {
	switch v := value.(type) {
	case bool:
		if v {
			return int64(1)
		}

		return int64(0)
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}

		return nil
	}

	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return int64(rv.Float())
	default:
		return nil
	}
}

// coerceReal maps a Go value to float64. It handles numeric strings and all
// signed/unsigned integer and float widths via reflection.
func coerceReal(value any) any {
	if v, ok := value.(string); ok {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}

		return nil
	}

	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	default:
		return nil
	}
}

// quoteIdent wraps a SQL identifier in double quotes, escaping any embedded
// double quotes by doubling them (SQL standard). This prevents SQL injection
// through user-declared field names used as column/table identifiers.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
