package sql

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// BuildWhereClause turns structured [kv.Condition]s into a parameterised WHERE
// clause (without the "WHERE" keyword). The placeholder function maps a 1-based
// parameter index to the driver-specific placeholder ($1, ?, :p1, …).
// Returns ("", nil) when conditions is empty.
func BuildWhereClause(conditions []kv.Condition, placeholder func(int) string) (string, []any) {
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

		if cond.Op == kv.OpIsNull || cond.Op == kv.OpIsNotNull {
			parts = append(parts, cond.Column+" "+string(cond.Op))

			continue
		}

		parts = append(parts, fmt.Sprintf("%s %s %s", cond.Column, cond.Op, placeholder(paramIdx)))
		args = append(args, cond.Value)
		paramIdx++
	}

	return strings.Join(parts, " AND "), args
}
