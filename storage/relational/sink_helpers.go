package relational

import (
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func partitionColumns(all, subset []string) ([]string, []string) {
	subsetSet := make(map[string]struct{}, len(subset))

	for _, c := range subset {
		subsetSet[c] = struct{}{}
	}

	var nonSubset, isSubset []string

	for _, c := range all {
		if _, ok := subsetSet[c]; ok {
			isSubset = append(isSubset, c)
		} else {
			nonSubset = append(nonSubset, c)
		}
	}

	return nonSubset, isSubset
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

// buildWhereClause turns structured Conditions into a parameterised WHERE
// clause (without the "WHERE" keyword). Returns ("", nil) when conditions is
// empty. Shared logic — mirrors the root storage package's buildWhereClause.
func buildWhereClause(conditions []kv.Condition, placeholder func(int) string) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(conditions))

	var args []any

	paramIdx := 1

	for _, cond := range conditions {
		if cond.Op == kv.OpIn {
			if len(cond.Values) == 0 {
				continue
			}

			placeholders := make([]string, 0, len(cond.Values))

			for range cond.Values {
				placeholders = append(placeholders, placeholder(paramIdx))
				paramIdx++
			}

			parts = append(parts, cond.Column+" IN ("+strings.Join(placeholders, ", ")+")")
			args = append(args, cond.Values...)

			continue
		}

		parts = append(parts, fmt.Sprintf("%s %s %s", cond.Column, cond.Op, placeholder(paramIdx)))
		args = append(args, cond.Value)
		paramIdx++
	}

	return strings.Join(parts, " AND "), args
}
