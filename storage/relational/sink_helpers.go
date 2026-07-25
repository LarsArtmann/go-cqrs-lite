package relational

import (
	"fmt"
	"sort"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func partitionColumns(all, subset []string) []string {
	subsetSet := make(map[string]struct{}, len(subset))

	for _, c := range subset {
		subsetSet[c] = struct{}{}
	}

	var nonSubset []string

	for _, c := range all {
		if _, ok := subsetSet[c]; !ok {
			nonSubset = append(nonSubset, c)
		}
	}

	return nonSubset
}

func excludedSet(cols []string) string {
	if len(cols) == 0 {
		return ""
	}

	parts := make([]string, len(cols))

	for i, c := range cols {
		parts[i] = c + " = excluded." + c
	}

	return strings.Join(parts, ", ")
}

func placeholders(dialect sqlpkg.Dialect, n int) string {
	ph := make([]string, n)

	for i := range n {
		ph[i] = dialect.Placeholder(i + 1)
	}

	return strings.Join(ph, ", ")
}

func eqWhere(cols []string, vals []any, dialect sqlpkg.Dialect, startIdx int) (string, []any) {
	parts := make([]string, len(cols))
	args := make([]any, len(cols))

	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s = %s", c, dialect.Placeholder(startIdx+i))
		args[i] = vals[i]
	}

	return strings.Join(parts, " AND "), args
}

// formatConditions returns a copy of conditions whose time.Time values are
// rendered through the dialect — so reads match the dialect-formatted
// timestamps the sink wrote. Without this, a WHERE created_at < ? bound with a
// raw time.Time would not compare correctly against TEXT-stored (SQLite) or
// TIMESTAMP-stored (Postgres) values.
func formatConditions(conditions []kv.Condition, dialect sqlpkg.Dialect) []kv.Condition {
	if len(conditions) == 0 {
		return conditions
	}

	out := make([]kv.Condition, len(conditions))

	for i, c := range conditions {
		c.Value = formatArg(c.Value, dialect)

		if len(c.Values) > 0 {
			vals := make([]any, len(c.Values))
			for j, v := range c.Values {
				vals[j] = formatArg(v, dialect)
			}

			c.Values = vals
		}

		out[i] = c
	}

	return out
}

func formatArg(v any, dialect sqlpkg.Dialect) any {
	if t, ok := v.(time.Time); ok {
		return dialect.FormatTime(t)
	}

	return v
}

// (buildWhereClause moved to storage/sql.BuildWhereClause — shared across
// relational and view sub-packages.)

// rowColumns turns a Row into sorted column names plus dialect-formatted
// values. Columns are sorted for deterministic SQL (stable golden output).
// Each column name is validated against the table's declared schema —
// unknown columns are rejected before they reach SQL.
func (s *sqlSink) rowColumns(table string, row Row) ([]string, []any, error) {
	if len(row) == 0 {
		return nil, nil, errSinkEmptyRow
	}

	t := s.schema.Table(table)
	if t == nil {
		return nil, nil, errorfamily.WrapRejection(errSinkUnknownTable,
			"relational.sink_unknown_table",
			fmt.Sprintf("table %q", table))
	}

	colSet := make(map[string]struct{}, len(t.Columns))
	for i := range t.Columns {
		colSet[t.Columns[i].Name] = struct{}{}
	}

	cols := make([]string, 0, len(row))
	for name := range row {
		if _, ok := colSet[name]; !ok {
			return nil, nil, errorfamily.WrapRejection(
				errSinkUnknownColumn,
				"relational.sink_unknown_column",
				fmt.Sprintf("table %q: column %q", table, name),
			)
		}

		cols = append(cols, name)
	}

	sort.Strings(cols)

	vals := make([]any, len(cols))
	for i, c := range cols {
		vals[i] = s.format(row[c])
	}

	return cols, vals, nil
}

func (s *sqlSink) format(v any) any {
	if t, ok := v.(time.Time); ok {
		return s.dialect.FormatTime(t)
	}

	return v
}

// conflictTarget returns the table's declared primary key, or []string{"id"}
// when no primary key is declared.
func (s *sqlSink) conflictTarget(table string) []string {
	t := s.schema.Table(table)
	if t == nil || len(t.PrimaryKey) == 0 {
		return []string{"id"}
	}

	return t.PrimaryKey
}
