package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// Count returns the number of records matching the query's filter, without
// loading any rows. This implements [kv.ViewCounter].
//
// When q.Where is empty, all records are counted. The OrderBy, Limit, and
// Offset fields of q are ignored — only Where and Args are used.
func (s *SQLViewStore[V, K]) Count(ctx context.Context, q kv.ViewQuery) (int64, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", s.mapper.Table)

	args := make([]any, 0, len(q.Args))

	if q.Where != "" {
		fmt.Fprintf(&b, " WHERE %s", q.Where)
		args = append(args, q.Args...)
	}

	var count int64

	err := s.DB.QueryRowContext(ctx, b.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("view-store: count: %w", err)
	}

	return count, nil
}

// QueryFiltered runs a structured (injection-safe) filter against the view
// table, combining it with the ordering and pagination from q. This implements
// [kv.FilteredQuerier].
//
// The filter's conditions are AND-joined and parameterised — no raw SQL from
// the caller touches the query string.
func (s *SQLViewStore[V, K]) QueryFiltered(
	ctx context.Context,
	f kv.ViewFilter,
	q kv.ViewQuery,
) ([]*V, error) {
	whereClause, args := buildWhereClause(f, s.Dialect.Placeholder)

	combined := kv.ViewQuery{
		Where:   whereClause,
		Args:    args,
		OrderBy: q.OrderBy,
		Desc:    q.Desc,
		Limit:   q.Limit,
		Offset:  q.Offset,
	}

	return s.Query(ctx, combined)
}

func buildWhereClause(f kv.ViewFilter, placeholder func(int) string) (string, []any) {
	if len(f.Conditions) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(f.Conditions))

	var args []any

	paramIdx := 1

	for _, cond := range f.Conditions {
		if cond.Op == kv.OpIn {
			values := toAnySlice(cond.Value)
			placeholders := make([]string, 0, len(values))
			for range values {
				placeholders = append(placeholders, placeholder(paramIdx))
				paramIdx++
			}

			parts = append(parts, cond.Column+" IN ("+strings.Join(placeholders, ", ")+")")
			args = append(args, values...)
			continue
		}

		parts = append(parts, fmt.Sprintf("%s %s %s", cond.Column, cond.Op, placeholder(paramIdx)))
		args = append(args, cond.Value)
		paramIdx++
	}

	return strings.Join(parts, " AND "), args
}

func toAnySlice(v any) []any {
	switch s := v.(type) {
	case []any:
		return s
	case []string:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	case []int:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	case []int64:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	case []uint64:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	case []float64:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	default:
		return []any{v}
	}
}
