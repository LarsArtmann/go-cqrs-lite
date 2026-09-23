package sql

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BuildWhereClause turns structured [kv.Condition]s into a parameterised WHERE
// clause (without the "WHERE" keyword). The placeholder function maps a 1-based
// parameter index to the driver-specific placeholder ($1, ?, :p1, …).
// Returns ("", nil) when conditions is empty.
//
// Deprecated: BuildWhereClause interpolates column names and operators
// without validation, so it is only safe for fully trusted, code-defined
// conditions. Use [BuildWhereClauseChecked], which rejects hostile column
// names and unsupported operators, whenever any part of a condition may
// originate outside your own code (e.g. HTTP query parameters).
func BuildWhereClause(conditions []kv.Condition, placeholder func(int) string) (string, []any) {
	clause, args, _ := BuildWhereClauseChecked(conditions, placeholder)
	return clause, args
}

// BuildWhereClauseChecked is [BuildWhereClause] with injection validation:
// every column must be a bare SQL identifier ([ValidateIdentifier]) and every
// operator must be supported ([ValidateOperator]). On violation it returns a
// descriptive error naming the offending condition and leaves the clause
// unbuilt — the caller must not run the query.
func BuildWhereClauseChecked(
	conditions []kv.Condition, placeholder func(int) string,
) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}

	for i, cond := range conditions {
		if !ValidateIdentifier(cond.Column) {
			return "", nil, errorfamily.NewRejection("storage.sql.invalid_column",
				fmt.Sprintf("condition %d: column %q is not a bare SQL identifier", i, cond.Column))
		}

		if !ValidateOperator(cond.Op) {
			return "", nil, errorfamily.NewRejection("storage.sql.invalid_operator",
				fmt.Sprintf("condition %d: unsupported operator %q", i, cond.Op))
		}
	}

	clause, args := buildWhereClause(conditions, placeholder)

	return clause, args, nil
}

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
