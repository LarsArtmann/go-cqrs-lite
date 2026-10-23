package metaengine

import (
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"time"
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
//
// Every column is TEXT: without the result type there is NO honest way to
// know a field's SQL type (the old field-NAME heuristic silently typed
// e.g. a "price" STRING column as REAL — corrupting future data on
// typed engines). For reflection-derived types from the stored struct,
// prefer [BuildLayoutPlanFromType].
func BuildLayoutPlan(collection string, filterFields, sortFields []string) LayoutPlan {
	return buildLayoutPlan(collection, filterFields, sortFields)
}

// buildLayoutPlan is the shared plan builder: dedup, deterministic column
// order, one index per column. Columns are TEXT — name-based type guessing
// is deleted (it silently mistyped future data); BuildLayoutPlanFromType
// overrides types it can derive from the result struct.
func buildLayoutPlan(
	collection string,
	filterFields, sortFields []string,
) LayoutPlan {
	table := "meta_planned_" + sanitize(collection)

	// Collect unique fields (dedup across filter + sort).
	seen := make(map[string]bool)

	var columns []PlannedColumn

	for _, f := range filterFields {
		if !seen[f] {
			seen[f] = true
			columns = append(columns, PlannedColumn{Name: f, Type: "TEXT"})
		}
	}

	for _, f := range sortFields {
		if !seen[f] {
			seen[f] = true
			columns = append(columns, PlannedColumn{Name: f, Type: "TEXT"})
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

	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", QuoteIdent(p.Table))
	b.WriteString("  key TEXT PRIMARY KEY,\n")
	b.WriteString("  value TEXT NOT NULL")

	for _, c := range p.Columns {
		fmt.Fprintf(&b, ",\n  %s %s", QuoteIdent(c.Name), c.Type)
	}

	b.WriteString("\n);")

	for _, idx := range p.Indexes {
		quotedCols := make([]string, len(idx.Columns))
		for i, col := range idx.Columns {
			quotedCols[i] = QuoteIdent(col)
		}

		colList := strings.Join(quotedCols, ", ")
		fmt.Fprintf(&b, "\nCREATE INDEX IF NOT EXISTS %s ON %s(%s);",
			QuoteIdent(idx.Name), QuoteIdent(p.Table), colList)
	}

	return b.String()
}

// QuoteIdent wraps a SQL identifier in double quotes, escaping any embedded
// double quotes by doubling them (SQL standard). This prevents SQL injection
// through user-declared field names used as column/table identifiers.
func QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
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

// sqlTypeOf maps a Go field type to its SQL column type. Kinds with a
// native mapping are typed; everything else degrades to TEXT (JSON still
// round-trips) and returns a non-nil advisory error so callers can warn —
// an unmapped type on a typed engine is a silent-corruption risk the
// operator should hear about.
func sqlTypeOf(t reflect.Type) (string, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER", nil
	case reflect.Float32:
		return "REAL", nil
	case reflect.Float64:
		return "DOUBLE", nil
	case reflect.Bool:
		return "INTEGER", nil
	case reflect.String:
		return "TEXT", nil
	}

	if t == reflect.TypeFor[time.Time]() {
		return "TEXT", nil
	}

	return "TEXT", fmt.Errorf(
		"no native SQL mapping for %s — column degrades to TEXT (JSON-encoded); "+
			"declare an explicit column type if typed scans must survive",
		t,
	)
}

// BuildLayoutPlanFromType is the reflection-based variant of BuildLayoutPlan:
// it infers each column's SQL type from the corresponding field of the result
// type R. Fields not present on R degrade to TEXT (name-based type guessing is
// deleted). This is the recommended constructor when the result type is known
// at plan time.
//
// Type mapping: int*/uint* → INTEGER, float32 → REAL, float64 → DOUBLE,
// bool → INTEGER, string → TEXT, time.Time → TEXT (ISO-8601). Everything else
// degrades to TEXT (with a warning) so JSON blobs still round-trip.
func BuildLayoutPlanFromType[R any](
	collection string,
	filterFields, sortFields []string,
) LayoutPlan {
	return BuildLayoutPlanForType(collection, filterFields, sortFields, reflect.TypeFor[R]())
}

// BuildLayoutPlanForType is the runtime-typed variant of BuildLayoutPlanFromType
// (same behavior; the result type arrives as a reflect.Type). A nil or
// non-struct resultType yields the all-TEXT plan of [BuildLayoutPlan].
func BuildLayoutPlanForType(
	collection string,
	filterFields, sortFields []string,
	resultType reflect.Type,
) LayoutPlan {
	typeOf := resultType
	fieldTypes := map[string]reflect.Type{}

	if typeOf != nil && typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}

	if typeOf != nil && typeOf.Kind() == reflect.Struct {
		for f := range typeOf.Fields() {
			if f.IsExported() {
				fieldTypes[f.Name] = f.Type
				// json-tag alias so filters on the wire name resolve to the
				// Go type (same resolution ExtractFields uses).
				if tag := JSONFieldName(f); tag != f.Name {
					fieldTypes[tag] = f.Type
				}
			}
		}
	}

	plan := BuildLayoutPlan(collection, filterFields, sortFields)

	for i, c := range plan.Columns {
		if t, ok := lookupFieldType(fieldTypes, c.Name); ok {
			colType, advisory := sqlTypeOf(t)
			plan.Columns[i].Type = colType

			if advisory != nil {
				slog.Warn(
					"metaengine: planned column type degraded to TEXT",
					"collection",
					collection,
					"column",
					c.Name,
					"go_type",
					t.String(),
					"because",
					advisory,
				)
			}
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

// BuildColumnarLayoutPlan creates a LayoutPlan that stores every exported field
// of the result type as a native SQL column. This is the "columnar-native"
// layout: the engine stores the value's fields directly as typed columns instead
// of a JSON blob, enabling vectorized scans and aggregations on columnar engines
// such as DuckDB. The key and value columns are still present so MapGet can
// reconstruct the original value without a full-column scan.
//
// Field names respect json tags (same as the engine's ExtractFields). When the
// result type is not a struct the plan contains only key and value columns.
func BuildColumnarLayoutPlan(collection string, resultType reflect.Type) LayoutPlan {
	table := "meta_planned_" + sanitize(collection)

	seen := make(map[string]bool)
	var columns []PlannedColumn

	if resultType.Kind() == reflect.Pointer {
		resultType = resultType.Elem()
	}

	if resultType.Kind() == reflect.Struct {
		for f := range resultType.Fields() {
			if !f.IsExported() {
				continue
			}

			name := JSONFieldName(f)
			if name == "" || seen[name] {
				continue
			}
			// Skip fields that collide with the base table columns (key, value).
			// DuckDB and SQLite are case-insensitive on unquoted identifiers,
			// so "Value" would collide with "value" and produce a DDL error.
			if lname := strings.ToLower(name); lname == "key" || lname == "value" {
				continue
			}
			seen[name] = true

			colType, advisory := sqlTypeOf(f.Type)

			if advisory != nil {
				slog.Warn(
					"metaengine: planned column type degraded to TEXT",
					"collection",
					collection,
					"column",
					name,
					"go_type",
					f.Type.String(),
					"because",
					advisory,
				)
			}

			columns = append(columns, PlannedColumn{Name: name, Type: colType})
		}
	}

	// Sort columns alphabetically for deterministic DDL.
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

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
