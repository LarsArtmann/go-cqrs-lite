package metaengine

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// LayoutPlan describes a planned table schema for a collection, optimizing
// query performance by extracting JSON fields into indexed columns.
//
// This is the Level 2 optimization (within-engine layout): instead of one
// meta_map table with json_extract() scans, the planner generates a dedicated
// table with extracted columns and indexes for declared filter/sort fields.
//
// The "Don't Be Stupid" rules:
// 1. One table + N indexes, not one projection per (filter, sort) combo.
// 2. Don't index a column never filtered.
// 3. Deduplicate indexes (RangeFilter + OrderBy on same column → one index).
type LayoutPlan struct {
	Collection string          // the collection name (query name)
	Table      string          // the table name (e.g., "meta_planned_users")
	Columns    []PlannedColumn // extracted columns from JSON
	Indexes    []PlannedIndex  // indexes on extracted columns
}

// PlannedColumn is an extracted column in a planned table.
type PlannedColumn struct {
	Name string // JSON field name (e.g., "status")
	Type string // SQL column type (e.g., "TEXT", "INTEGER")
}

// PlannedIndex is an index on one or more planned columns.
type PlannedIndex struct {
	Name    string   // index name (e.g., "idx_planned_users_status")
	Columns []string // column names
}

// BuildLayoutPlan creates a LayoutPlan from a query's declared filter/sort
// fields. It follows the "Don't Be Stupid" rules:
//   - Only fields declared via FilterOnField or SortOnField get extracted columns
//   - Duplicate fields (filtered + sorted) get ONE column, not two
//   - Each unique column gets one index
func BuildLayoutPlan(collection string, filterFields, sortFields []string) LayoutPlan {
	table := "meta_planned_" + sanitize(collection)

	// Collect unique fields (dedup across filter + sort).
	seen := make(map[string]bool)

	var columns []PlannedColumn

	for _, f := range filterFields {
		if !seen[f] {
			seen[f] = true
			columns = append(columns, PlannedColumn{Name: f, Type: inferColumnType(f)})
		}
	}

	for _, f := range sortFields {
		if !seen[f] {
			seen[f] = true
			columns = append(columns, PlannedColumn{Name: f, Type: inferColumnType(f)})
		}
	}

	// Sort columns alphabetically for deterministic DDL.
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	// One index per column (rule 3: dedup).
	indexes := make([]PlannedIndex, 0, len(columns))

	for _, c := range columns {
		indexes = append(indexes, PlannedIndex{
			Name:    fmt.Sprintf("idx_%s_%s", table, c.Name),
			Columns: []string{c.Name},
		})
	}

	return LayoutPlan{
		Collection: collection,
		Table:      table,
		Columns:    columns,
		Indexes:    indexes,
	}
}

// ColumnNames returns the names of all planned columns in order.
func (p LayoutPlan) ColumnNames() []string {
	names := make([]string, len(p.Columns))
	for i, c := range p.Columns {
		names[i] = c.Name
	}
	return names
}

// DDL generates the CREATE TABLE and CREATE INDEX statements for this plan.
func (p LayoutPlan) DDL() string {
	var b strings.Builder

	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", p.Table)
	b.WriteString("  key TEXT PRIMARY KEY,\n")
	b.WriteString("  value TEXT NOT NULL")

	for _, c := range p.Columns {
		fmt.Fprintf(&b, ",\n  %s %s", c.Name, c.Type)
	}

	b.WriteString("\n);")

	for _, idx := range p.Indexes {
		colList := strings.Join(idx.Columns, ", ")
		fmt.Fprintf(&b, "\nCREATE INDEX IF NOT EXISTS %s ON %s(%s);",
			idx.Name, p.Table, colList)
	}

	return b.String()
}

// sanitize makes a collection name safe for use as a SQL identifier.
func sanitize(s string) string {
	var b strings.Builder

	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteRune('_')
		}
	}

	return b.String()
}

// inferColumnType guesses a SQL column type from a field name.
// inferColumnType guesses a SQL column type from a field name.
// This is intentionally conservative — TEXT works for all JSON types.
// Prefer BuildLayoutPlanFromType for reflection-based inference from the result
// type's Go fields, which is accurate instead of name-heuristic.
func inferColumnType(field string) string {
	field = strings.ToLower(field)

	// Common numeric field names.
	switch {
	case strings.Contains(field, "priority"),
		strings.Contains(field, "count"),
		strings.Contains(field, "age"),
		strings.Contains(field, "level"),
		strings.Contains(field, "score"),
		strings.Contains(field, "num"),
		strings.Contains(field, "amount"),
		strings.Contains(field, "price"):
		return "INTEGER"
	default:
		return "TEXT"
	}
}

// BuildLayoutPlanFromType is the reflection-based variant of BuildLayoutPlan:
// it infers each column's SQL type from the corresponding field of the result
// type R, instead of guessing from the field name. Fields not present on R fall
// back to the name heuristic (TEXT-safe). This is the recommended constructor
// when the result type is known at plan time.
//
// Type mapping: int*/float* → INTEGER/REAL, bool → INTEGER, string → TEXT,
// time.Time → TEXT (ISO-8601). Everything else defaults to TEXT so JSON blobs
// still round-trip.
func BuildLayoutPlanFromType[R any](
	collection string,
	filterFields, sortFields []string,
) LayoutPlan {
	typeOf := reflect.TypeFor[R]()
	fieldTypes := map[string]reflect.Type{}

	if typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}

	if typeOf.Kind() == reflect.Struct {
		for f := range typeOf.Fields() {
			if f.IsExported() {
				fieldTypes[f.Name] = f.Type
			}
		}
	}

	plan := BuildLayoutPlan(collection, filterFields, sortFields)

	for i, c := range plan.Columns {
		if t, ok := lookupFieldType(fieldTypes, c.Name); ok {
			plan.Columns[i].Type = sqlTypeOf(t)
		}
	}

	return plan
}

func lookupFieldType(fields map[string]reflect.Type, name string) (reflect.Type, bool) {
	for fname, t := range fields {
		if strings.EqualFold(fname, name) {
			return t, true
		}
	}

	return nil, false
}

func sqlTypeOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "INTEGER"
	default:
		return "TEXT"
	}
}
