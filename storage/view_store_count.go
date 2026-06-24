package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// Count returns the number of records matching the query's conditions, without
// loading any rows. This implements [kv.ViewCounter].
//
// When q.Conditions is empty, all records are counted. The OrderBy, Limit, and
// Offset fields of q are ignored — only Conditions are used.
func (s *SQLViewStore[V, K]) Count(ctx context.Context, q kv.ViewQuery) (int64, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", s.mapper.Table)

	whereClause, args := buildWhereClause(q.Conditions, s.Dialect.Placeholder)

	if whereClause != "" {
		fmt.Fprintf(&b, " WHERE %s", whereClause)
	}

	var count int64

	err := s.DB.QueryRowContext(ctx, b.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("view-store: count: %w", err)
	}

	return count, nil
}

// buildWhereClause turns structured Conditions into a parameterised WHERE
// clause (without the "WHERE" keyword). Returns ("", nil) when conditions is
// empty.
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
