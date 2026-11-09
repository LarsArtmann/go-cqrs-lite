package metaengine

import (
	"fmt"
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
